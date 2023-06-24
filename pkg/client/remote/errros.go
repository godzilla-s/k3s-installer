package remote

import "errors"

var (
	ErrK3SNotRunning = errors.New("k3s is not running")
	ErrFileExist     = errors.New("file has been exist")
)
