package vm

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Docker helpers
//
// All Docker interaction shells out to the `docker` CLI via os/exec, following
// the ACI handler pattern. No Docker SDK dependency is added.
// ---------------------------------------------------------------------------

// checkDocker returns true when the Docker CLI is reachable and the daemon is
// responsive. Unlike the ACI probe it does not require /var/run/docker.sock to
// exist, so Docker Desktop on Windows/macOS is detected too; the LookPath
// pre-check keeps startup fast when the CLI is absent.
func checkDocker() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	out, err := exec.Command("docker", "info").CombinedOutput()
	if err != nil {
		log.Printf("[vm] docker info failed: %v – %s", err, strings.TrimSpace(string(out)))
		return false
	}
	return true
}

// runOpts describes a container to start with dockerRun.
type runOpts struct {
	name    string
	image   string
	env     []string // KEY=VALUE pairs
	ports   []int    // published as host:container with the same number
	network string   // optional Docker network to join
	cmd     []string // optional command override after the image
}

// dockerRun creates and starts a container, returning its ID. The host
// gateway alias is always added so containers can reach the miniblue API via
// host.docker.internal on Linux as well.
func dockerRun(o runOpts) (string, error) {
	args := []string{"run", "-d", "--name", o.name,
		"--add-host", "host.docker.internal:host-gateway"}
	if o.network != "" {
		args = append(args, "--network", o.network)
	}
	for _, e := range o.env {
		args = append(args, "-e", e)
	}
	for _, p := range o.ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", p, p))
	}
	// "--" terminates flag parsing so a user-supplied image or command that
	// starts with "-" is treated as a positional arg, not a docker run flag.
	args = append(args, "--", o.image)
	args = append(args, o.cmd...)

	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker run failed: %v – %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// dockerInspectRunning reports whether the named container is running.
func dockerInspectRunning(name string) bool {
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", name).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// dockerInspectExit returns the exit code and error string of a stopped
// container. ok is false when the container cannot be inspected at all.
func dockerInspectExit(name string) (code int, errMsg string, ok bool) {
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.ExitCode}}|{{.State.Error}}", name).CombinedOutput()
	if err != nil {
		return 0, "", false
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 2)
	code, _ = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		errMsg = parts[1]
	}
	return code, errMsg, true
}

func dockerStart(name string) error {
	out, err := exec.Command("docker", "start", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker start failed: %v – %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func dockerStop(name string) error {
	out, err := exec.Command("docker", "stop", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker stop failed: %v – %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// dockerRemove force-removes a container. Errors are logged only: removal is
// used in cleanup paths that must not fail the ARM-level operation.
func dockerRemove(name string) {
	out, err := exec.Command("docker", "rm", "-f", name).CombinedOutput()
	if err != nil {
		log.Printf("[vm] docker rm %s: %v – %s", name, err, strings.TrimSpace(string(out)))
	}
}

func dockerNetworkCreate(name string) error {
	out, err := exec.Command("docker", "network", "create", name).CombinedOutput()
	if err != nil {
		// Treat an already-existing network as success (idempotent PUT).
		if strings.Contains(string(out), "already exists") {
			return nil
		}
		return fmt.Errorf("docker network create failed: %v – %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func dockerNetworkRemove(name string) {
	out, err := exec.Command("docker", "network", "rm", name).CombinedOutput()
	if err != nil {
		log.Printf("[vm] docker network rm %s: %v – %s", name, err, strings.TrimSpace(string(out)))
	}
}

// dockerExec runs a script inside a running container through `sh -c`, so
// shell semantics (env expansion, pipes) apply. Returns combined output and
// the command's exit code.
func dockerExec(name string, script string) (output string, exitCode int, err error) {
	out, err := exec.Command("docker", "exec", name, "sh", "-c", script).CombinedOutput()
	if err != nil {
		if ee, isExit := err.(*exec.ExitError); isExit {
			return string(out), ee.ExitCode(), nil
		}
		return string(out), -1, err
	}
	return string(out), 0, nil
}

// dockerLogsOutput returns a snapshot of a container's logs (stdout+stderr).
func dockerLogsOutput(name string, tail int) (string, error) {
	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "--tail", strconv.Itoa(tail))
	}
	args = append(args, name)
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %v – %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// dockerLogsFollow starts a streaming `docker logs -f` and returns a reader of
// the combined stdout+stderr plus a stop function that kills the process.
func dockerLogsFollow(name string, tail int) (io.ReadCloser, func(), error) {
	args := []string{"logs", "-f"}
	if tail > 0 {
		args = append(args, "--tail", strconv.Itoa(tail))
	}
	args = append(args, name)
	cmd := exec.Command("docker", args...)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return nil, nil, err
	}
	go func() {
		cmd.Wait()
		pw.Close()
	}()
	stop := func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
	return pr, stop, nil
}
