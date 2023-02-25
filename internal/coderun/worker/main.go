package main

import (
	"context"
	"fmt"
	"io"
	"mock-server/internal/coderun/scripts"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"net/http"
	"os"
	"strings"

	zlog "github.com/rs/zerolog/log"
)

//// format
// Headers:
// - run_type -- {mapper, dyn_handle}
// - script
// Body
// - json -- {arg: argval}

func parseRequest(req *http.Request) (*scripts.RunRequest, error) {
	var run_request scripts.RunRequest

	for name, headers := range req.Header {
		if strings.EqualFold(name, "RunType") {
			run_request.RunType = headers[0]
		} else if strings.EqualFold(name, "Script") {
			run_request.Script = headers[0]
		}
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	run_request.Args = body

	return &run_request, nil
}

func runHandle(w http.ResponseWriter, req *http.Request) {
	zlog.Info().Msg("processing request")
	run_request, err := parseRequest(req)
	if err != nil {
		zlog.Error().Err(err).Msg("parse error")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "parse error: %s", err.Error())
		return
	}

	run_ctx, cancel := context.WithTimeout(req.Context(), configs.GetCoderunConfig().WorkerConfig.HandleTimeout)
	defer cancel()

	out, err := scripts.RunPythonScript(run_ctx, run_request)
	if err != nil {
		zlog.Error().Err(err).Msg("run error")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "run error: %s", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, out)
}

func main() {
	// load config
	configs.LoadConfig()

	// init logger
	logger.Init(configs.GetLogConfig())

	port, ok := os.LookupEnv("PORT")
	if !ok {
		zlog.Error().Msg("port wasn't provided to worker")
		os.Exit(1)
	}

	// set port variable for every log msg
	zlog.Logger = zlog.Logger.With().Str("port", port).Logger()

	http.HandleFunc("/run", runHandle)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)

	zlog.Error().Err(err).Msg("server start failed")
	panic(err)
}
