package muxr

import "github.com/go-logr/logr"

// A multiplexing logr implementation

type muxr []logr.LogSink

func NewMuxLogger(loggers ...logr.Logger) logr.Logger {
	theMuxr := make(muxr, 0, len(loggers))
	for i := range loggers {
		sink := loggers[i].GetSink()
		if sinkWCallDepth, ok := sink.(logr.CallDepthLogSink); ok {
			// Skip the muxr methods when looking up caller info
			sink = sinkWCallDepth.WithCallDepth(1)
		}
		theMuxr = append(theMuxr, sink)
	}
	return logr.Logger{}.WithSink(theMuxr)
}

// Helper to force us to implement the CallDepthLogSink interface
var _ logr.CallDepthLogSink = muxr{}

func (mx muxr) Init(info logr.RuntimeInfo) {
	for _, m := range mx {
		m.Init(info)
	}
}

func (mx muxr) Enabled(level int) (ena bool) {
	for _, m := range mx {
		ena = ena || m.Enabled(level)
	}
	return
}

func (mx muxr) Info(level int, msg string, keysAndValues ...any) {
	for _, m := range mx {
		m.Info(level, msg, keysAndValues...)
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
		if mWCallDepth, ok := m.(logr.CallDepthLogSink); ok {
			m = mWCallDepth.WithCallDepth(depth)
		}
		newMuxr = append(newMuxr, m)
	}
	return newMuxr
}
