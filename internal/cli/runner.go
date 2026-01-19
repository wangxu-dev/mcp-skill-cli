package cli

import (
	"io"
	"time"
)

func RunWithSpinner(out io.Writer, message string, tips []string, threshold time.Duration, fn func() error) error {
	if threshold <= 0 {
		return fn()
	}

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	timer := time.NewTimer(threshold)
	defer timer.Stop()

	select {
	case err := <-done:
		return err
	case <-timer.C:
		spinner := StartSpinnerWithTips(out, message, tips)
		err := <-done
		spinner.Stop()
		return err
	}
}
