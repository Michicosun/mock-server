package docker

import (
	"bytes"
	"context"
	"fmt"
	"mock-server/internal/configs"
	"mock-server/internal/util"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	zlog "github.com/rs/zerolog/log"
)

type DockerProvider struct {
	ctx       context.Context
	cli       *client.Client
	cfg       *configs.ContainerResources
	tagPrefix string
}

func NewDockerProvider(ctx context.Context, tagPrefix string, cfg *configs.ContainerResources) (*DockerProvider, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker adapter")
	}

	return &DockerProvider{
		ctx:       ctx,
		cli:       cli,
		cfg:       cfg,
		tagPrefix: tagPrefix,
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
			if strings.HasPrefix(tag, dp.tagPrefix) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (dp *DockerProvider) PruneWorkerImages() ([]types.ImageDeleteResponseItem, error) {
	has_worker_images, err := dp.hasWorkerImages()
	if err != nil {
		return make([]types.ImageDeleteResponseItem, 0), err
	}
	if has_worker_images {
		zlog.Info().Msg("worker image exists on host, prunning")
		return dp.cli.ImageRemove(dp.ctx, dp.tagPrefix, types.ImageRemoveOptions{
			PruneChildren: true,
			Force:         true,
		})
	}
	zlog.Info().Msg("worker image doesn't exitst on host")
	return make([]types.ImageDeleteResponseItem, 0), nil
}

func (dp *DockerProvider) BuildWorkerImage() error {
	_, err := dp.PruneWorkerImages()
	if err != nil {
		return errors.Wrap(err, "prune old images")
	}

	root, err := util.GetProjectRoot()
	if err != nil {
		return err
	}

	dockerfile_path, err := filepath.Abs(filepath.Join(root, "internal", "coderun", "worker", "Dockerfile"))
	if err != nil {
		return err
	}

	zlog.Info().Str("path", dockerfile_path).Msg("using dockerfile")

	cmd := exec.Command("docker", "build", ".", "-f", dockerfile_path, "-t", dp.tagPrefix)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, errb.String())
	}

	return nil
}

func (dp *DockerProvider) CreateWorkerContainer(port string) (string, error) {
	zlog.Info().Str("port", port).Msg("creating worker container")

	cport := fmt.Sprintf("%s/tcp", port)

	contConfig := &container.Config{
		Image: dp.tagPrefix,
		ExposedPorts: nat.PortSet{
			nat.Port(cport): struct{}{},
		},
	}

	mount_list := make([]mount.Mount, 0)

	if configs.GetLogConfig().FileLoggingEnabled {
		mount_list = append(mount_list, mount.Mount{
			Type:   mount.TypeBind,
			Source: configs.GetLogConfig().Directory,
			Target: configs.GetLogConfig().Directory,
		})
	}

	// TODO add code directory to mount list

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			NanoCPUs: int64(dp.cfg.CPULimit * 1e9),
			Memory:   int64(dp.cfg.MemoryLimit * 1e6),
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
	l.Msg("container created")

	return cont.ID, nil
}

func (dp *DockerProvider) StartWorkerContainer(id string) error {
	return dp.cli.ContainerStart(dp.ctx, id, types.ContainerStartOptions{})
}

func (dp *DockerProvider) RemoveWorkerContainer(id string, force bool) error {
	zlog.Info().Str("id", id).Msg("removing container")
	return dp.cli.ContainerRemove(dp.ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         force,
	})
}
