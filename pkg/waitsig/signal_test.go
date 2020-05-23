package waitsig

import (
	"syscall"
	"testing"
	"time"
)

func TestWaitSignal(t *testing.T) {
	stop := make(chan struct{})

	go func() {
		time.Sleep(1e4)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	WaitSignal(stop)
	<-stop
}
