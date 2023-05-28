package remote

import "errors"

var (
	ErrK3SNotRunning = errors.New("k3s is not running")
	ErrFileDoesExist = errors.New("file does exist")
)
