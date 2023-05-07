package coderun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mock-server/internal/coderun/docker-provider"
	"mock-server/internal/configs"
	"mock-server/internal/util"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	zlog "github.com/rs/zerolog/log"
)

var WorkerWatcher = &watcher{}

type worker struct {
	watcher *watcher
	port    string
	cId     string
}

type Args struct {
	args [][]byte
}

func NewDynHandleArgs(args []byte) *Args {
	return &Args{
		args: [][]byte{args},
	}
}

func NewMapperArgs(msgs []string) *Args {
	args := &Args{
		args: make([][]byte, 0),
	}

	for _, msg := range msgs {
		args.args = append(args.args, []byte(msg))
	}

	return args
}

func (w *worker) RunScript(run_type string, script string, args *Args) ([]byte, error) {
	zlog.Info().Str("run_type", run_type).Str("script", script).Msg("preparing worker request")

	var byteArgs []byte
	switch run_type {
	case "dyn_handle":
		byteArgs = args.args[0]
	case "mapper":
		byteArgs = util.WrapArgsForEsb(args.args)

	default:
		return nil, fmt.Errorf("invalid run type: %s. Expected `dyn_handle` or `mapper`", run_type)
	}

	bodyReader := bytes.NewReader(byteArgs)
	requestURL := fmt.Sprintf("http://127.0.0.1:%s/run", w.port)
	ctx, cancel := context.WithTimeout(w.watcher.ctx, configs.GetCoderunConfig().WorkerConfig.HandleTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Add("RunType", run_type)
	req.Header.Add("Script", script)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	out, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("run script error: %s, code: %d", out, res.StatusCode)
	}

	return out, err
}

func (w *worker) Return() {
	w.watcher.workers <- w
}

type watcher struct {
	wg           sync.WaitGroup
	ctx          context.Context
	dp           *docker.DockerProvider
	workers      chan *worker
	repair       chan *worker
	workerByPort map[string]*worker
}

func getFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port), nil
}

func (w *watcher) startNewWorker() error {
	zlog.Info().Msg("starting new worker")
	port, err := getFreePort()
	if err != nil {
		return err
	}

	id, err := w.dp.CreateWorkerContainer(port)
	if err != nil {
		return err
	}

	if err := w.dp.StartWorkerContainer(id); err != nil {
		return err
	}

	worker := worker{
		watcher: w,
		port:    port,
		cId:     id,
	}

	w.workers <- &worker
	w.workerByPort[port] = &worker

	return nil
}

func (w *watcher) processWorkerInfo(worker *worker, info *types.ContainerJSON) {
	if info.State.Running {
		w.workers <- worker
	}
	if info.State.Paused || info.State.OOMKilled || info.State.Dead {
		w.dp.RestartWorkerContainer(worker.cId) // nolint:errcheck
		w.repair <- worker
	}
	// info.State.Restarting
}

func (w *watcher) repairLoop() {
	for {
		select {
		case <-w.ctx.Done():
			w.wg.Done()
			return
		case worker := <-w.repair:
			zlog.Info().Str("port", worker.port).Msg("repairing worker")
			info, err := w.dp.InspectWorkerContainer(worker.cId)
			if err != nil {
				w.dp.RemoveWorkerContainer(worker.cId, true) // nolint:errcheck
				delete(w.workerByPort, worker.port)
				w.startNewWorker() // nolint:errcheck
			} else {
				w.processWorkerInfo(worker, &info)
			}

			time.Sleep(1 * time.Second) // TODO fix sleep time
		}
	}
}

func (w *watcher) Init(ctx context.Context, cfg *configs.CoderunConfig) error {
	zlog.Info().Msg("starting watcher")

	w.ctx = ctx

	provider, err := docker.NewDockerProvider(ctx, &cfg.WorkerConfig.ContainerConfig)
	if err != nil {
		return err
	}

	w.dp = provider

	err = w.dp.BuildWorkerImage()
	if err != nil {
		return err
	}

	w.workers = make(chan *worker, cfg.WorkerCnt)
	w.repair = make(chan *worker, cfg.WorkerCnt)
	w.workerByPort = make(map[string]*worker)
	for i := 0; i < cfg.WorkerCnt; i += 1 {
		err = w.startNewWorker()
		if err != nil {
			w.Stop()
			return err
		}
	}

	// wait until containers start working
	time.Sleep(2 * time.Second)

	w.wg.Add(1)
	go w.repairLoop() // start repair loop

	return nil
}

func (w *watcher) BorrowWorker() (*worker, error) {
	zlog.Info().Msg("trying to borrow worker")
	ctx, cancel := context.WithTimeout(w.ctx, configs.GetCoderunConfig().WorkerConfig.HandleTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("no running worker now")
		case worker := <-w.workers:
			info, err := w.dp.InspectWorkerContainer(worker.cId)
			if err != nil {
				return nil, err
			}

			if info.State.Running {
				return worker, nil
			}

			zlog.Info().Str("port", worker.port).Str("state", info.State.Status).Msg("found broken worker")
			w.repair <- worker
		}
	}
}

func (w *watcher) Stop() {
	zlog.Info().Msg("stopping watcher")
	w.wg.Wait()
	close(w.repair)
	close(w.workers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.dp.ChangeContext(ctx)

	for _, worker := range w.workerByPort {
		w.dp.RemoveWorkerContainer(worker.cId, true) // nolint:errcheck
	}

	zlog.Info().Msg("watcher stopped")
}
