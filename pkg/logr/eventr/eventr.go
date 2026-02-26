package eventr

import (
	"fmt"
	"slices"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
)

const (
	MaxLoggingLevel = 0
	ReasonLogKey    = "eventReason"
)

// A logr implementation generating K8s events (events API)
type eventr struct {
	recorder      events.EventRecorder
	subject       runtime.Object
	keysAndValues []any
}

func NewEventr(recorder events.EventRecorder, subject runtime.Object) logr.Logger {
	return logr.Logger{}.WithSink(&eventr{recorder: recorder, subject: subject})
}

func (r *eventr) Init(info logr.RuntimeInfo) {}

func (r *eventr) Enabled(level int) bool {
	return level <= MaxLoggingLevel
}

func (r *eventr) Info(level int, msg string, keysAndValues ...any) {
	reason := GetValueForKey(keysAndValues, ReasonLogKey, GetValueForKey(r.keysAndValues, ReasonLogKey, "Info"))
	r.recorder.Eventf(r.subject, nil, "Normal", reason, "Info", msg)
}

func (r *eventr) Error(err error, msg string, keysAndValues ...any) {
	reason := GetValueForKey(keysAndValues, ReasonLogKey, GetValueForKey(r.keysAndValues, ReasonLogKey, "Info"))
	r.recorder.Eventf(r.subject, nil, "Warning", reason, "Info", "%s: %s", msg, err.Error())
}

func (r *eventr) WithValues(keysAndValues ...any) logr.LogSink {
	return &eventr{
		recorder:      r.recorder,
		subject:       r.subject,
		keysAndValues: append(keysAndValues, r.keysAndValues...),
	}
}

func (r *eventr) WithName(name string) logr.LogSink {
	return r
}

func GetValueForKey(keysAndValues []any, key any, defVal string) string {
	value := defVal
	if idx := slices.Index(keysAndValues, key); idx > -1 && idx+1 < len(keysAndValues) {
		switch rawVal := keysAndValues[idx+1].(type) {
		case string:
			value = rawVal
		case fmt.Stringer:
			value = rawVal.String()
		}
	}
	return value
}
