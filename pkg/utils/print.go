package utils

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

type Print struct {
	step atomic.Int32
	mux  sync.Mutex
}

func NewMessage() *Print {
	return &Print{}
}

func (p *Print) Step(format string, v ...any) {
	p.step.Add(1)
	fmt.Fprintln(os.Stdout, fmt.Sprintf("Step %d: %s", p.step.Load(), fmt.Sprintf(format, v...)))
}

func (p *Print) Message(format string, v ...any) {
	fmt.Fprintln(os.Stdout, fmt.Sprintf("==> %s", fmt.Sprintf(format, v...)))
}

func (p *Print) Error(format string, v ...any) {
	fmt.Fprintln(os.Stdout, fmt.Sprintf("==> %s", fmt.Sprintf(format, v...)))
}

func (p *Print) Warn(format string, v ...any) {
	fmt.Fprintln(os.Stdout, fmt.Sprintf("==> %s", fmt.Sprintf(format, v...)))
}
