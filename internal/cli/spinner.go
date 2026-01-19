package cli

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type Spinner struct {
	out      io.Writer
	message  string
	tips     []string
	stop     chan struct{}
	done     chan struct{}
	lastLen  int
	lastTip  time.Time
	tipIndex int
}

func StartSpinner(out io.Writer, message string) *Spinner {
	return startSpinner(out, message, nil)
}

func StartSpinnerWithTips(out io.Writer, message string, tips []string) *Spinner {
	return startSpinner(out, message, tips)
}

func startSpinner(out io.Writer, message string, tips []string) *Spinner {
	if out == nil {
		return &Spinner{done: make(chan struct{})}
	}
	spinner := &Spinner{
		out:     out,
		message: message,
		tips:    sanitizeTips(tips),
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
			line := s.formatLine(frame)
			if s.lastLen > len(line) {
				line += strings.Repeat(" ", s.lastLen-len(line))
			}
			fmt.Fprintf(s.out, "\r%s", line)
			s.lastLen = len(line)
		}
	}
}

func (s *Spinner) clear() {
	if s.lastLen == 0 {
		fmt.Fprint(s.out, "\r \r")
		return
	}
	padding := strings.Repeat(" ", s.lastLen)
	fmt.Fprintf(s.out, "\r%s\r", padding)
	s.lastLen = 0
}

func (s *Spinner) formatLine(frame rune) string {
	tip := s.nextTip()
	switch {
	case s.message != "" && tip != "":
		return fmt.Sprintf("%c %s - %s", frame, s.message, tip)
	case s.message != "":
		return fmt.Sprintf("%c %s", frame, s.message)
	case tip != "":
		return fmt.Sprintf("%c %s", frame, tip)
	default:
		return fmt.Sprintf("%c", frame)
	}
}

func (s *Spinner) nextTip() string {
	if len(s.tips) == 0 {
		return ""
	}
	now := time.Now()
	if s.lastTip.IsZero() {
		s.lastTip = now
		return s.tips[s.tipIndex%len(s.tips)]
	}
	if now.Sub(s.lastTip) >= 2*time.Second {
		s.tipIndex = (s.tipIndex + 1) % len(s.tips)
		s.lastTip = now
	}
	return s.tips[s.tipIndex%len(s.tips)]
}

func sanitizeTips(tips []string) []string {
	var cleaned []string
	for _, tip := range tips {
		tip = strings.TrimSpace(tip)
		if tip == "" {
			continue
		}
		cleaned = append(cleaned, tip)
	}
	return cleaned
}

func DefaultTips() []string {
	return []string{
		"Please wait a moment. This can take a little while.",
		"Working on it. You can keep this window open.",
		"Still running... almost there.",
		"Tip: Use --available to browse registry entries.",
		"Tip: Use --global or --local to choose install scope.",
		"Tip: Use --client to target specific apps.",
		"Tip: Use update to refresh installed items.",
	}
}
