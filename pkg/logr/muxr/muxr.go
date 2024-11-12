package muxr

import "github.com/go-logr/logr"

// A multiplexing logr implementation

type muxr []logr.Logger

func NewMuxLogger(loggers ...logr.Logger) logr.Logger {
	theMuxr := make(muxr, 0, len(loggers))
	for i := range loggers {
		// We skip 2 call stack frames because the Logger is calling our sink
		// which then calls the underlying logger.
		theMuxr = append(theMuxr, loggers[i].WithCallDepth(2))
	}
	return logr.Logger{}.WithSink(theMuxr)
}

// Helper to force us to implement the CallDepthLogSink interface
var _ logr.CallDepthLogSink = muxr{}

func (mx muxr) Init(info logr.RuntimeInfo) {
	for i := range mx {
		mx[i].GetSink().Init(info)
	}
}

func (mx muxr) Enabled(level int) (ena bool) {
	for i := range mx {
		ena = ena || mx[i].GetSink().Enabled(level)
	}
	return
}

func (mx muxr) Info(level int, msg string, keysAndValues ...any) {
	for _, m := range mx {
		m.V(level-m.GetV()).Info(msg, keysAndValues...)
	}
}

func (mx muxr) Error(err error, msg string, keysAndValues ...any) {
	for _, m := range mx {
		m.Error(err, msg, keysAndValues...)
	}
}

func (mx muxr) WithValues(keysAndValues ...any) logr.LogSink {
	newMuxr := make(muxr, 0, len(mx))
	for _, m := range mx {
		newMuxr = append(newMuxr, m.WithValues(keysAndValues...))
	}
	return newMuxr
}

func (mx muxr) WithName(name string) logr.LogSink {
	newMuxr := make(muxr, 0, len(mx))
	for _, m := range mx {
		newMuxr = append(newMuxr, m.WithName(name))
	}
	return newMuxr
}

func (mx muxr) WithCallDepth(depth int) logr.LogSink {
	newMuxr := make(muxr, 0, len(mx))
	for _, m := range mx {
		m = m.WithCallDepth(depth)
		newMuxr = append(newMuxr, m)
	}
	return newMuxr
}
