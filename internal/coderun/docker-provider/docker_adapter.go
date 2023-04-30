package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mock-server/internal/configs"
	"mock-server/internal/util"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
)

const (
	TAG_PREFIX = "mock-server-coderun-worker"
)

type DockerProvider struct {
	ctx context.Context
	cli *client.Client
	cfg *configs.ContainerConfig
}

func NewDockerProvider(ctx context.Context, cfg *configs.ContainerConfig) (*DockerProvider, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker adapter")
	}

	return &DockerProvider{
		ctx: ctx,
		cli: cli,
		cfg: cfg,
	}, nil
}

func (dp *DockerProvider) Close() {
	dp.cli.Close()
}

func (dp *DockerProvider) hasWorkerImages() (bool, error) {
	images, err := dp.cli.ImageList(dp.ctx, types.ImageListOptions{
		All: true,
	})

	if err != nil {
		return false, err
	}

	for i := 0; i < len(images); i++ {
		for _, tag := range images[i].RepoTags {
			if strings.HasPrefix(tag, TAG_PREFIX) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (dp *DockerProvider) ChangeContext(ctx context.Context) {
	dp.ctx = ctx
}

func (dp *DockerProvider) PruneWorkerImages() error {
	zlog.Info().Msg("removing worker image")
	_, err := dp.cli.ImageRemove(dp.ctx, TAG_PREFIX, types.ImageRemoveOptions{
		PruneChildren: true,
		Force:         true,
	})
	return err
}

type errorDetail struct {
	Message string `json:"message"`
}

type errorLine struct {
	Error       string      `json:"error"`
	ErrorDetail errorDetail `json:"errorDetail"`
}

func parseDockerBuildLogs(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		zlog.Debug().Msg(lastLine)
	}

	errLine := &errorLine{}
	if err := json.Unmarshal([]byte(lastLine), errLine); err != nil {
		return err
	}

	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (dp *DockerProvider) BuildWorkerImage() error {
	zlog.Info().Msg("building worker image")

	has_worker_images, err := dp.hasWorkerImages()
	if err != nil {
		return err
	}

	if has_worker_images {
		if dp.cfg.NeedRebuild {
			if err := dp.PruneWorkerImages(); err != nil {
				return errors.Wrap(err, "prune old images")
			}
		} else {
			zlog.Info().Msg("image already exists on host and rebuild is disabled")
			return nil
		}
	}

	root, err := util.GetProjectRoot()
	if err != nil {
		return err
	}

	dockerfile_path := filepath.Join("internal", "coderun", "worker", "Dockerfile")
	zlog.Info().Str("path", dockerfile_path).Msg("using dockerfile")

	tar, err := archive.TarWithOptions(root, &archive.TarOptions{})
	if err != nil {
		return err
	}

	defer tar.Close()

	opts := types.ImageBuildOptions{
		Dockerfile: dockerfile_path,
		Tags:       []string{TAG_PREFIX},
		Remove:     true,
	}

	res, err := dp.cli.ImageBuild(dp.ctx, tar, opts)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err = parseDockerBuildLogs(res.Body); err != nil {
		return err
	}

	return nil
}

func getEnvList(port string) []string {
	return []string{
		fmt.Sprintf("PORT=%s", port),
	}
}

func getMountList() ([]mount.Mount, error) {
	file_storage_root, err := util.FileStorageRoot()
	if err != nil {
		return nil, err
	}

	mount_list := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: file_storage_root,
			Target: "/worker_dir/.storage",
		},
	}

	if configs.GetLogConfig().FileLoggingEnabled {
		mount_list = append(mount_list, mount.Mount{
			Type:   mount.TypeBind,
			Source: configs.GetLogConfig().Directory,
			Target: configs.GetLogConfig().Directory,
		})
	}

	return mount_list, nil
}

func (dp *DockerProvider) CreateWorkerContainer(port string) (string, error) {
	zlog.Info().Str("port", port).Msg("creating worker container")

	cport := fmt.Sprintf("%s/tcp", port)

	contConfig := &container.Config{
		Image: TAG_PREFIX,
		ExposedPorts: nat.PortSet{
			nat.Port(cport): struct{}{},
		},
		Env: getEnvList(port),
	}

	mount_list, err := getMountList()
	if err != nil {
		return "", err
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			NanoCPUs: int64(dp.cfg.CPULimit * 1e9),
			Memory:   int64(dp.cfg.MemoryLimitMB * 1e6),
		},
		PortBindings: nat.PortMap{
			nat.Port(cport): []nat.PortBinding{
				{
					HostIP:   "127.0.0.1", // traffic only from loopback to worker
					HostPort: port,
				},
			},
		},
		Mounts: mount_list,
	}

	cont, err := dp.cli.ContainerCreate(dp.ctx, contConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", errors.Wrap(err, "container cannot be created")
	}

	l := zlog.Info()
	if len(cont.Warnings) != 0 {
		l = zlog.Warn().Strs("warnings", cont.Warnings)
	}

	l.Str("id", cont.ID).Msg("worker container created")
	return cont.ID, nil
}

func (dp *DockerProvider) StartWorkerContainer(id string) error {
	zlog.Info().Str("id", id).Msg("starting worker container")
	return dp.cli.ContainerStart(dp.ctx, id, types.ContainerStartOptions{})
}

func (dp *DockerProvider) InspectWorkerContainer(id string) (types.ContainerJSON, error) {
	zlog.Info().Str("id", id).Msg("inspecting worker container")
	return dp.cli.ContainerInspect(dp.ctx, id)
}

func (dp *DockerProvider) StopWorkerContainer(id string) error {
	zlog.Info().Str("id", id).Msg("stopping worker container")
	return dp.cli.ContainerStop(dp.ctx, id, container.StopOptions{})
}

func (dp *DockerProvider) RestartWorkerContainer(id string) error {
	zlog.Info().Str("id", id).Msg("restarting worker container")
	return dp.cli.ContainerRestart(dp.ctx, id, container.StopOptions{})
}

func (dp *DockerProvider) RemoveWorkerContainer(id string, force bool) error {
	zlog.Info().Str("id", id).Msg("removing worker container")
	return dp.cli.ContainerRemove(dp.ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         force,
	})
}
