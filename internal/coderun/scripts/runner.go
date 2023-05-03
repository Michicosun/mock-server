package scripts

import (
	"bytes"
	"context"
	"fmt"
	"mock-server/internal/util"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
)

type RunRequest struct {
	RunType string
	Script  string
	Args    []byte
}

func getCoderunRoot() (string, error) {
	root, err := util.FileStorageRoot()
	if err != nil {
		return "", err
	}

	coderun_root, err := filepath.Abs(filepath.Join(root, "coderun"))
	if err != nil {
		return "", err
	}

	return coderun_root, nil
}

func RunPythonScript(ctx context.Context, req *RunRequest) (string, error) {
	zlog.Info().Str("type", req.RunType).Str("script", req.Script).Str("args", string(req.Args)).Msg("running script")
	coderun_root, err := getCoderunRoot()
	if err != nil {
		return "", errors.Wrap(err, "get coderun root failed")
	}

	script_full_path := filepath.Join(coderun_root, req.RunType, req.Script)
	if _, err := os.Stat(script_full_path); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("script: %s not exists", script_full_path)
	}

	if err = os.WriteFile("data.json", req.Args, 0644); err != nil {
		return "", errors.Wrap(err, "dump args to file failed")
	}

	cmd := exec.CommandContext(ctx, "python3", script_full_path)
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		zlog.Error().Str("type", req.RunType).Str("script", req.Script).Err(err).Msg("run failed")
		return stderr.String(), err
	}

	zlog.Info().Str("type", req.RunType).Str("script", req.Script).Msg("successfully finished")
	return stdout.String(), nil
}
