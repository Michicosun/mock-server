package coderun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mock-server/internal/coderun/docker-provider"
	"mock-server/internal/configs"
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

func (w *worker) RunScript(run_type string, script string, args interface{}) ([]byte, error) {
	zlog.Info().Str("run_type", run_type).Str("script", script).Msg("preparing worker request")
	jsonBody, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	bodyReader := bytes.NewReader(jsonBody)
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
	if err != nil {
		return nil, err
	}

	return out, err
}

func (w *worker) Return() {
	w.watcher.workers <- w
}

type watcher struct {
	initialized bool
	wg          sync.WaitGroup
	ctx         context.Context
	dp          *docker.DockerProvider
	workers     chan *worker
	repair      chan *worker
	portworker  map[string]*worker
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

	w.dp.StartWorkerContainer(id)
	if err != nil {
		return err
	}

	worker := worker{
		watcher: w,
		port:    port,
		cId:     id,
	}

	w.workers <- &worker
	w.portworker[port] = &worker

	return nil
}

func (w *watcher) processWorkerInfo(worker *worker, info *types.ContainerJSON) {
	if info.State.Running {
		w.workers <- worker
	}
	if info.State.Paused || info.State.OOMKilled || info.State.Dead {
		w.dp.RestartWorkerContainer(worker.cId)
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
				w.dp.RemoveWorkerContainer(worker.cId, true)
				delete(w.portworker, worker.port)
				w.startNewWorker()
			} else {
				w.processWorkerInfo(worker, &info)
			}

			time.Sleep(1 * time.Second) // TODO fix sleep time
		}
	}
}

func (w *watcher) Init(ctx context.Context, cfg *configs.CoderunConfig) error {
	zlog.Info().Msg("initializing watcher")
	if w.initialized {
		return fmt.Errorf("watcher has already initialized")
	}
	w.initialized = true
	w.ctx = ctx

	provider, err := docker.NewDockerProvider(ctx, &cfg.WorkerConfig.Resources)
	if err != nil {
		return err
	}

	w.dp = provider

	err = w.dp.BuildWorkerImage()
	if err != nil {
		return err
	}

	w.workers = make(chan *worker, configs.GetCoderunConfig().WorkerCnt)
	w.repair = make(chan *worker, configs.GetCoderunConfig().WorkerCnt)
	w.portworker = make(map[string]*worker)
	for i := 0; i < configs.GetCoderunConfig().WorkerCnt; i += 1 {
		err = w.startNewWorker()
		if err != nil {
			return err
		}
	}

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

	for _, worker := range w.portworker {
		w.dp.RemoveWorkerContainer(worker.cId, true)
	}

	zlog.Info().Msg("watcher stopped")
}