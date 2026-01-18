package cli

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type Spinner struct {
	out     io.Writer
	message string
	stop    chan struct{}
	done    chan struct{}
}

func StartSpinner(out io.Writer, message string) *Spinner {
	if out == nil {
		return &Spinner{done: make(chan struct{})}
	}
	spinner := &Spinner{
		out:     out,
		message: message,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go spinner.run()
	return spinner
}

func (s *Spinner) Stop() {
	if s == nil {
		return
	}
	if s.stop == nil {
		return
	}
	close(s.stop)
	<-s.done
}

func (s *Spinner) run() {
	defer close(s.done)
	frames := []rune{'|', '/', '-', '\\'}
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	index := 0
	for {
		select {
		case <-s.stop:
			s.clear()
			return
		case <-ticker.C:
			frame := frames[index%len(frames)]
			index++
			if s.message == "" {
				fmt.Fprintf(s.out, "\r%c", frame)
				continue
			}
			fmt.Fprintf(s.out, "\r%c %s", frame, s.message)
		}
	}
}

func (s *Spinner) clear() {
	if s.message == "" {
		fmt.Fprint(s.out, "\r \r")
		return
	}
	padding := strings.Repeat(" ", len(s.message)+2)
	fmt.Fprintf(s.out, "\r%s\r", padding)
}
