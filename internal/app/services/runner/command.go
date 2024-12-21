package runner

import (
	"go.uber.org/zap"
	"os"
	"os/exec"
	"stream-recorder/pkg/logger"
	"strings"
)

type Command struct{}

func NewCmd() *Command {
	return &Command{}
}

func (c *Command) RunCommands(commands [][]string, ignoreErr bool) error {
	for _, args := range commands {
		logger.Debug("Executing command", zap.String("cmd", strings.Join(args, " ")))
		cmd := exec.Command(args[0], args[1:]...)
		err := c.RunCommand(cmd, ignoreErr)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Command) RunCommand(cmd *exec.Cmd, ignoreErr bool) error {
	cmd.SysProcAttr = GetSysProcAttr()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil && !ignoreErr {
		logger.Error("Command execution failed", zap.String("cmd", cmd.String()), zap.Error(err))
		return err
	}

	return nil
}
