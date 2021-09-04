package internal

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
)

type BlindNginxRestartAction struct{}

func CmdCheck(ctx context.Context, name string, args ...string) (tryNext bool, err1 error) {
	exe, err := exec.LookPath("exe")
	if err == nil {
		cmd := exec.CommandContext(ctx, exe, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err == nil {
			return false, nil
		}
		if err == context.Canceled || err == context.DeadlineExceeded {
			return false, err
		}
		return true, err
	}
	return false, nil
}

func (b *BlindNginxRestartAction) Do(ctx context.Context) error {
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
	if _, err = os.Stat("/var/run/nginx.pid"); os.IsNotExist(err) {
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
