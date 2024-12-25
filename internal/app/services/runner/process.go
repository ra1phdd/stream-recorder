package runner

import (
	"bufio"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"runtime"
	"stream-recorder/pkg/logger"
	"syscall"
)

type Process struct{}

func NewProcess() *Process {
	return &Process{}
}

func (p *Process) Run(name string, cmd *exec.Cmd, fileName, output string, handler func(line string), waitForExit func(cmd *exec.Cmd, fileName, output string)) error {
	logger.Info("Starting process", zap.String("name", name))

	cmd.SysProcAttr = GetSysProcAttr()
	cmd.Stderr = os.Stderr

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("Failed to get stdout pipe", zap.Error(err))
		return err
	}

	if err := cmd.Start(); err != nil {
		logger.Error("Failed to start command", zap.Error(err))
		return err
	}

	go p.handleStdout(name, stdoutPipe, handler)
	go waitForExit(cmd, fileName, output)

	logger.Debug("process started successfully", zap.String("name", name))
	return nil
}

func (p *Process) handleStdout(name string, stdoutPipe io.ReadCloser, handler func(line string)) {
	logger.Info(fmt.Sprintf("Handling stdout for %s", name))

	scanner := bufio.NewScanner(stdoutPipe)
	defer func(stdoutPipe io.ReadCloser) {
		err := stdoutPipe.Close()
		if err != nil {
			logger.Error("Failed to close stdout pipe", zap.Error(err))
			return
		}
	}(stdoutPipe)

	for scanner.Scan() {
		line := scanner.Text()
		logger.Info(fmt.Sprintf("Received line from %s", name), zap.String("line", line))
		handler(line)
	}
}

func (p *Process) Kill(name string, cmd *exec.Cmd) error {
	logger.Info("Stopping process", zap.String("name", name))

	if cmd != nil && cmd.Process != nil {
		var err error
		if runtime.GOOS == "windows" {
			err = cmd.Process.Kill()
		} else {
			err = cmd.Process.Signal(syscall.SIGTERM)
		}
		if err != nil {
			logger.Error("Failed to kill process", zap.Error(err))
			return err
		}

		logger.Debug("Process killed")
	}

	logger.Debug("Process stopped successfully")
	return nil
}

func (p *Process) Terminate(name string) error {
	logger.Info("Terminate processes", zap.String("name", name))

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("taskkill", "/FI", fmt.Sprintf("IMAGENAME eq %s*", name), "/F")
	} else {
		cmd = exec.Command("pkill", "-f", name)
	}

	output, err := cmd.CombinedOutput()
	if err != nil && err.Error() != "exit status 1" {
		logger.Error("Failed to terminate processes", zap.Error(err), zap.String("output", string(output)))
		return err
	}

	logger.Info("Successfully terminated processes")
	return nil
}
