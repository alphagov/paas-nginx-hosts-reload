package utils

import "time"

type Utils interface {
	ModTime() (time.Time, error)
	SigHup(pid int32) error
	FindNginxPID(string, string) (int32, error)
}
