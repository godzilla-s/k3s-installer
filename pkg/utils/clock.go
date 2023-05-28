package utils

import (
	"fmt"
	"time"
)

func Clock(timeout, interval time.Duration, process func() error) error {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout")
		case <-ticker.C:
			if err := process(); err == nil {
				return nil
			}
		}
	}
}
