package elog

import (
	"github.com/jbenet/go-ipfs/Godeps/_workspace/src/code.google.com/p/go.net/context"
	logging "github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-logging"
	"github.com/jbenet/go-ipfs/util"
)

var eloggers = map[string]*logging.Logger{}

func init() {
	SetupLogging()
}

type EventLogger interface {
	StandardLogger
	Event(ctx context.Context, event string, m ...Loggable)
}

type StandardLogger interface {
	Critical(args ...interface{})
	Criticalf(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Notice(args ...interface{})
	Noticef(format string, args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
}

// Logger retrieves a particular event logger
func Logger(system string) EventLogger {
	return &eventLogger{util.Logger(system)}
}

// eventLogger implements the EventLogger and wraps a go-logging Logger
type eventLogger struct {
	*logging.Logger
}

func (el *eventLogger) Event(ctx context.Context, event string, metadata ...Loggable) {
	existing, err := MetadataFromContext(ctx)
	if err != nil {
		existing = Metadata{}
	}
	accum := existing
	for _, datum := range metadata {
		accum = DeepMerge(accum, datum.Loggable())
	}
	accum["event"] = event

	str, err := accum.JsonString()
	if err != nil {
		return
	}
	el.Logger.Info(str)
}

// SetupLogging will initialize the logger backend and set the flags.
func SetupLogging() {
	// 	fmt := logging.DefaultFormatter

	// 	f, err := os.Create("events.ipfslog")
	// 	if err != nil {
	// 		panic("failed to open file for event logger")
	// 	}
}
