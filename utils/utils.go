package utils

import (
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/process"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Utils

type utils struct {
}

func NewUtils() *utils {
	return &utils{}
}

func (u *utils) ModTime() (time.Time, error) {
	currentStat, err := os.Stat("/etc/hosts")
	if err != nil {
		return time.Time{}, err
	}
	return currentStat.ModTime(), nil
}

func (u *utils) SigHup(pid int32) error {
	return syscall.Kill(int(pid), syscall.SIGHUP)
}

func (u *utils) FindNginxPID(processName string, cmdSubstring string) (int32, error) {
	processes, err := process.Processes()
	if err != nil {
		return 0, err
	}
	for _, p := range processes {
		if name, _ := p.Name(); name == processName {
			if cmdline, _ := p.Cmdline(); strings.Contains(cmdline, cmdSubstring) {
				pid := p.Pid
				return pid, nil
			}
		}
	}
	return 0, nil
}
