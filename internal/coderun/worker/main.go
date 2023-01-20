package main

import (
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/logger"
	"mock-server/internal/util"
	"net/http"
	"os"
	"os/exec"

	zlog "github.com/rs/zerolog/log"
)

func run(w http.ResponseWriter, req *http.Request) {
	cmd := exec.Command("python3", "--version")
	stdout, err := cmd.Output()
	if err != nil {
		zlog.Error().Err(err).Msg("python3")
	}

	fmt.Fprintf(w, "python3 version: %s\n", string(stdout))

	driver, err := util.NewFileStorageDriver("coderun")
	if err != nil {
		zlog.Error().Err(err).Msg("driver initialization")
	}

	s, err := driver.Read("mappers", "a")
	if err != nil {
		zlog.Error().Err(err).Msg("read")
	}

	zlog.Info().Msg("got request")
	fmt.Fprintf(w, "hello: %s\n", s)
}

func headers(w http.ResponseWriter, req *http.Request) {
	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func main() {
	// load config
	configs.LoadConfig()

	// init logger
	logger.Init(configs.GetLogConfig())

	port, ok := os.LookupEnv("PORT")
	if !ok {
		os.Exit(1)
	}

	// set port variable for every log msg
	zlog.Logger = zlog.Logger.With().Str("port", port).Logger()

	http.HandleFunc("/run", run)
	http.HandleFunc("/headers", headers)

	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
