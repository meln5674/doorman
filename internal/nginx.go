package internal

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type BlindNginxRestartAction struct{}

func FindExe(options ...string) string {
	for _, exe := range options {
		if path, err := exec.LookPath(exe); err == nil {
			return path
		}
	}
	return ""
}

func CmdCheck(ctx context.Context, name string, args ...string) (tryNext bool, err1 error) {
	exe, err := exec.LookPath(name)
	if err == nil {
		cmd := exec.CommandContext(ctx, exe, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("> %s\n", strings.Join(append([]string{name}, args...), " "))
		err := cmd.Run()
		if err == nil {
			return false, nil
		}
		if err == context.Canceled || err == context.DeadlineExceeded {
			return false, err
		}
		return true, err
	} else {
		fmt.Printf("No such command: %s\n", name)
		return false, nil
	}
}

func (b *BlindNginxRestartAction) Do(ctx context.Context) error {

	become := FindExe("sudo", "doas")
	restarter := FindExe("systemctl", "service", "docker")

	if restarter != "" {
		if become != "" {
			tryNext, err := CmdCheck(ctx, become, "systemctl", "restart", "nginx")
			if !tryNext {
				return err
			}
			tryNext, err = CmdCheck(ctx, become, "service", "nginx", "restart")
			if !tryNext {
				return err
			}
			tryNext, err = CmdCheck(ctx, become, "docker", "restart", "nginx")
			if !tryNext {
				return err
			}
		}
		tryNext, err := CmdCheck(ctx, "systemctl", "restart", "nginx")
		if !tryNext {
			return err
		}
		tryNext, err = CmdCheck(ctx, "service", "nginx", "restart")
		if !tryNext {
			return err
		}
		tryNext, err = CmdCheck(ctx, "docker", "restart", "nginx")
		if !tryNext {
			return err
		}

	}

	if _, err := os.Stat("/var/run/nginx.pid"); os.IsNotExist(err) {
		return fmt.Errorf("Ran out of ideas for restarting nginx")
	}
	pidFileBytes, err := ioutil.ReadFile("/var/run/nginx.pid")
	if err != nil {
		return err
	}
	pidFileString := string(pidFileBytes)
	pid, err := strconv.Atoi(pidFileString)
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(unix.SIGHUP)
}
