package logger

import (
	"sync"
	"testing"
)

var waitGroup sync.WaitGroup

func loopgetLogger(t *testing.T, routineNum int) {
	defer waitGroup.Done()
	for i := 0; i < 1000; i++ {
		logger1 := GetLogger()
		if logger1 == nil {
			t.Errorf("GetLogger() = nil in goroutine %d, expected a non-nil logger", routineNum)
		}
	}

}
func TestGetLogger(t *testing.T) {
	if GetLogger() == nil {
		t.Errorf("GetLogger() = nil, expected a non-nil logger")
	}

	waitGroup.Add(2)
	go loopgetLogger(t, 1)
	go loopgetLogger(t, 2)
	waitGroup.Wait()
}
