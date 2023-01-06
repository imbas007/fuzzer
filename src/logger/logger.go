package logger

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log *zap.Logger

	red        = color.New(color.FgRed)
	boldRed    = red.Add(color.Bold)
	white      = color.New(color.FgWhite)
	boldWhite  = white.Add(color.Bold)
	background = boldRed.Add(color.BgHiWhite)
)

func init() {
	Log, _ = Setup(true)
	defer Log.Sync()
}

// configure will return instance of zap logger configuration, configured to be verbose or to use JSON formatting
func Setup(verbose bool) (logger *zap.Logger, err error) {
	level := zapcore.InfoLevel
	if verbose {
		level = zapcore.DebugLevel
	}

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "console",
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:    "message",
			LevelKey:      "level",
			TimeKey:       "time",
			NameKey:       "logger",
			CallerKey:     "go",
			StacktraceKey: "trace",
			LineEnding:    "\n",
			EncodeLevel:   zapcore.CapitalColorLevelEncoder,
			// EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
				now := time.Now()
				out := now.Format("02.01.2006 15:04:05.99")
				out = fmt.Sprintf("[ %s ]", out)

				enc.AppendString(out)
			},
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller: func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
				callerName := caller.TrimmedPath()
				callerName = minWidth(callerName, " ", 20)
				enc.AppendString(callerName)
			},
			EncodeName: zapcore.FullNameEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: nil,
		InitialFields:    nil,
	}

	return config.Build()
}
