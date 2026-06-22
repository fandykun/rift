package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type terminalSpinner struct {
	writer  io.Writer
	message string
	active  bool
	done    chan struct{}
	once    sync.Once
	mu      sync.Mutex
}

func startTerminalSpinner(writer io.Writer, message string) *terminalSpinner {
	spinner := &terminalSpinner{writer: writer, message: message}
	if !isCharacterDevice(writer) {
		return spinner
	}

	spinner.active = true
	spinner.done = make(chan struct{})
	go spinner.run()
	return spinner
}

func (s *terminalSpinner) stop() {
	if !s.active {
		return
	}
	s.once.Do(func() {
		close(s.done)
	})
}

func (s *terminalSpinner) printLine(format string, args ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		clearTerminalLine(s.writer)
	}
	fmt.Fprintf(s.writer, format, args...)
}

func (s *terminalSpinner) run() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		clearTerminalLine(s.writer)
	}()

	index := 0
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			fmt.Fprintf(s.writer, "\r%s %s", frames[index%len(frames)], s.message)
			s.mu.Unlock()
			index++
		case <-s.done:
			return
		}
	}
}

func isCharacterDevice(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func clearTerminalLine(writer io.Writer) {
	fmt.Fprintf(writer, "\r%s\r", strings.Repeat(" ", 96))
}
