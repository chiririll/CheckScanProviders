package nativelog

import (
	"fmt"
	"os"
	"sync"
)

type Sink func(prio int, msg string)

var (
	sinkMu sync.Mutex
	sink   Sink
)

func SetSink(fn Sink) {
	sinkMu.Lock()
	sink = fn
	sinkMu.Unlock()
}

func write(prio int, msg string) {
	sinkMu.Lock()
	fn := sink
	sinkMu.Unlock()
	if fn != nil {
		fn(prio, msg)
		return
	}
	label := "I"
	switch prio {
	case prioDebug:
		label = "D"
	case prioWarn:
		label = "W"
	case prioError:
		label = "E"
	}
	fmt.Fprintf(os.Stderr, "checkscan %s %s\n", label, msg)
}
