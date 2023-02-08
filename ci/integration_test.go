package ci_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NginxHostsReloadIntegration", func() {
	var (
		cli  *client.Client
		ctx  context.Context
		resp container.ContainerCreateCreatedBody
	)
	BeforeEach(func() {
		var err error
		ctx = context.Background()

		dockerSockPath := filepath.Join(os.Getenv("HOME"), ".docker/run/docker.sock")
		if _, err = os.Stat(dockerSockPath); err == nil {

			cli, err = client.NewClient("unix://"+dockerSockPath, "", nil, nil)
		} else {
			cli, err = client.NewClientWithOpts(client.FromEnv)
		}
		Expect(err).NotTo(HaveOccurred())

		hostConfig := &container.HostConfig{
			PortBindings: map[nat.Port][]nat.PortBinding{
				"80/tcp": {{HostPort: "8081"}},
			},
		}

		resp, err = cli.ContainerCreate(ctx, &container.Config{
			Image: "nginx-hosts-reload:latest",
		}, hostConfig, nil, nil, "")

		Expect(err).NotTo(HaveOccurred())
		Expect(err).NotTo(HaveOccurred())
		Expect(cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})).To(Succeed())

		eventuallyRespondsOnPort("localhost:8081")
	})

	It("should not update the endpoint on a host file change without nginx-hosts-reload running", func() {

		webResp, err := http.Get("http://localhost:8081")
		Expect(err).NotTo(HaveOccurred())
		Expect(webResp.StatusCode).To(Equal(200))

		_, rc, err := RunDockerCommand(cli, ctx, resp.ID, []string{"/bin/bash", "-c", "sed 's/127.0.0.1 testserver/8.8.8.8 testserver/g' /etc/hosts > /tmp/hosts && cp /tmp/hosts /etc/hosts"})
		Expect(err).NotTo(HaveOccurred())
		Expect(rc).To(Equal(0))

		// Be consistent with other test
		time.Sleep(2 * time.Second)

		webResp, err = http.Get("http://localhost:8081")
		Expect(err).NotTo(HaveOccurred())
		Expect(webResp.StatusCode).To(Equal(200))
	})

	It("should update the endpoint on a host file change without nginx-hosts-reload running", func() {

		// Execute a command in the running container
		_, rc, err := RunDockerCommand(cli, ctx, resp.ID, []string{"/bin/bash", "-c", "nohup nginx-hosts-reload --interval 1s > /dev/null 2>&1 &"})
		Expect(err).NotTo(HaveOccurred())
		Expect(rc).To(Equal(0))

		time.Sleep(1 * time.Second)

		webResp, err := http.Get("http://localhost:8081")
		Expect(err).NotTo(HaveOccurred())
		Expect(webResp.StatusCode).To(Equal(200))

		_, rc, err = RunDockerCommand(cli, ctx, resp.ID, []string{"/bin/bash", "-c", "sed 's/127.0.0.1 testserver/172.17.0.6 testserver/g' /etc/hosts > /tmp/hosts && cp /tmp/hosts /etc/hosts"})
		Expect(err).NotTo(HaveOccurred())
		Expect(rc).To(Equal(0))

		// Give nginx-hosts-reload time to reload the config
		time.Sleep(2 * time.Second)

		webResp, err = http.Get("http://localhost:8081")
		Expect(err).NotTo(HaveOccurred())
		Expect(webResp.StatusCode).To(Equal(502))
	})

	AfterEach(func() {
		// Stop and remove the container
		Expect(cli.ContainerStop(ctx, resp.ID, nil)).To(Succeed())
		Expect(cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})).To(Succeed())
	})
})

func RunDockerCommand(cli *client.Client, ctx context.Context, containerID string, cmd []string) (string, int, error) {
	exec, err := cli.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", 0, err
	}

	resp, err := cli.ContainerExecAttach(ctx, exec.ID, types.ExecStartCheck{})
	if err != nil {
		resp.Close()
		return "", 0, err
	}
	defer resp.Close()

	// Read the output
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Reader)
	if err != nil {
		return "", 0, err
	}

	inspect, err := cli.ContainerExecInspect(ctx, exec.ID)
	if err != nil {
		return "", 0, err
	}

	return buf.String(), inspect.ExitCode, nil
}

func eventuallyRespondsOnPort(hostport string) {
	Eventually(func() error {
		_, err := http.Get("http://localhost:8081")
		if err != nil {
			fmt.Println("Port not up yet")
			return err
		}
		return nil
	}, 2*time.Minute, 10*time.Second).Should(Succeed())
}
