package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/fatih/color"
	pLog "github.com/phuslu/log"
	"github.com/samber/lo"
)

// levelColorMap represents mapping between log level string and it's version for console writer.
var levelColorMap = map[string]string{
	"trace": color.BlueString("TRACE"),
	"debug": color.BlueString("DEBUG"),
	"info":  color.GreenString("INFO"),
	"warn":  color.YellowString("WARN"),
	"error": color.RedString("ERROR"),
	"fatal": color.RedString("FATAL"),
	"panic": color.RedString("PANIC"),
}

// keyValue represents phuslu log key/value pair type alias.
type keyValue = struct {
	Key       string
	Value     string
	ValueType byte
}

// Level represents log level.
type Level uint32

const (
	TraceLevel Level = Level(pLog.TraceLevel) // 1
	DebugLevel Level = Level(pLog.DebugLevel) // 2
	InfoLevel  Level = Level(pLog.InfoLevel)  // 3
	WarnLevel  Level = Level(pLog.WarnLevel)  // 4
	ErrorLevel Level = Level(pLog.ErrorLevel) // 5
	FatalLevel Level = Level(pLog.FatalLevel) // 6
	PanicLevel Level = Level(pLog.PanicLevel) // 7
)

// Logger represents wrapper over logging library.
type Logger struct {
	writer *pLog.MultiEntryWriter
	*pLog.Logger
}

// New returns new configured logger with log level `lvl`.
func New(lvl Level) *Logger {
	writer := pLog.MultiEntryWriter{
		&pLog.ConsoleWriter{
			Formatter: newConsoleFormatter(true, time.DateTime),
			Writer:    os.Stderr,
		},
	}
	log := pLog.Logger{
		Level:  pLog.Level(lvl),
		Writer: &writer,
	}
	return &Logger{Logger: &log, writer: &writer}
}

// Trace prints trace level `msg` with caller.
func (l Logger) Trace(msg any) {
	l.Logger.Trace().Caller(2).Msg(fmt.Sprint(msg))
}

// Tracef prints trace level message from `args` in given `format`.
func (l Logger) Tracef(format string, args ...any) {
	l.Logger.Trace().Caller(2).Msgf(format, args...)
}

// TraceFi prints trace level `msg` with caller and formatted and colored `fields`.
func (l Logger) TraceFi(msg string, fields ...any) {
	print(l.Logger.Trace().Caller(2), msg, fields)
}

// Debug prints debug level `msg` with caller.
func (l Logger) Debug(msg any) {
	l.Logger.Debug().Caller(2).Msg(fmt.Sprint(msg))
}

// Debugf prints debug level message from `args` in given `format`.
func (l Logger) Debugf(format string, args ...any) {
	l.Logger.Debug().Caller(2).Msgf(format, args...)
}

// DebugFi prints debug level `msg` with caller and formatted and colored `fields`.
func (l Logger) DebugFi(msg string, fields ...any) {
	print(l.Logger.Debug().Caller(2), msg, fields)
}

// Info prints info level `msg`.
func (l Logger) Info(msg any) {
	l.Logger.Info().Msg(fmt.Sprint(msg))
}

// Infof prints info level message from `args` in given `format`.
func (l Logger) Infof(format string, args ...any) {
	l.Logger.Info().Msgf(format, args...)
}

// InfoFi prints info level `msg` with formatted and colored `fields`.
func (l Logger) InfoFi(msg string, fields ...any) {
	print(l.Logger.Info(), msg, fields)
}

// Warn prints warning level `msg`.
func (l Logger) Warn(msg any) {
	l.Logger.Warn().Msg(fmt.Sprint(msg))
}

// Warnf prints warning level message from `args` in given `format`.
func (l Logger) Warnf(format string, args ...any) {
	l.Logger.Warn().Msgf(format, args...)
}

// WarnFi prints warning level `msg` with formatted and colored `fields`.
func (l Logger) WarnFi(msg string, fields ...any) {
	print(l.Logger.Warn(), msg, fields)
}

// Error prints error level `msg`.
func (l Logger) Error(msg any) {
	l.Logger.Error().Msg(fmt.Sprint(msg))
}

// Errorf prints error level message from `args` in given `format`.
func (l Logger) Errorf(format string, args ...any) {
	l.Logger.Error().Msgf(format, args...)
}

// ErrorFi prints error level `msg` with formatted and colored `fields`.
func (l Logger) ErrorFi(msg string, fields ...any) {
	print(l.Logger.Error(), msg, fields)
}

// Fatal prints fatal level `msg` and exits the program.
func (l Logger) Fatal(msg any) {
	l.Logger.Fatal().Msg(fmt.Sprint(msg))
}

// Fatalf prints fatal level message from `args` in given `format` and exits the program.
func (l Logger) Fatalf(format string, args ...any) {
	l.Logger.Fatal().Msgf(format, args...)
}

// FatalFi prints fatal level `msg` with formatted and colored `fields` and exits the program.
func (l Logger) FatalFi(msg string, fields ...any) {
	print(l.Logger.Fatal(), msg, fields)
}

// Panic prints panic level `msg` and panics.
func (l Logger) Panic(msg any) {
	l.Logger.Panic().Msg(fmt.Sprint(msg))
}

// Panicf prints panic level message from `args` in given `format` and panics.
func (l Logger) Panicf(format string, args ...any) {
	l.Logger.Panic().Msgf(format, args...)
}

// PanicFi prints panic level `msg` with formatted and colored `fields` and panics.
func (l Logger) PanicFi(msg string, fields ...any) {
	print(l.Logger.Panic(), msg, fields)
}

// AddFileWriter creates log file at `filePath` and adds file writer to logger.
//
// If `filePath` is empty string, it does nothing and returns nil error.
func (l Logger) AddFileWriter(filePath string) (*os.File, error) {
	if filePath == "" {
		return nil, nil
	}

	logFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "Open or create log file")
	}
	*l.writer = append(*l.writer, &pLog.IOWriter{
		Writer: logFile,
	})

	return logFile, nil
}

// print adds message `msg` and `fields` to `entry` and prints it.
func print(entry *pLog.Entry, msg string, fields []any) {
	// Not using entry.KeysAndValues() as it will not add keys which can't be converted to string by type assertion
	var key string
	for i, field := range fields {
		if i%2 == 0 {
			key = fmt.Sprint(field)
		} else {
			entry.Any(key, field)
		}
	}
	entry.Msg(msg)
}

// newConsoleFormatter returns formatter funtion with `timeFormat` for console writer.
//
// If `colorize` is true, add colors to output.
func newConsoleFormatter(colorize bool, timeFormat string) func(io.Writer, *pLog.FormatterArgs) (int, error) {
	return func(w io.Writer, a *pLog.FormatterArgs) (int, error) {
		gray := color.RGB(118, 118, 118).SprintFunc()
		var messageSb strings.Builder

		formatterTime, err := time.Parse(time.RFC3339Nano, a.Time) // Get time object from FormatterArgs
		if err != nil {
			return 0, err
		}
		properTime := formatterTime.Format(timeFormat)
		if colorize {
			messageSb.WriteString(gray(properTime))
		} else {
			messageSb.WriteString(properTime)
		}
		messageSb.WriteRune(' ')

		if colorize {
			messageSb.WriteString(levelColorMap[a.Level])
		} else {
			messageSb.WriteString(strings.ToUpper(a.Level))
		}
		messageSb.WriteRune(' ')

		if a.Caller != "" {
			if colorize {
				messageSb.WriteString(gray(a.Caller))
			} else {
				messageSb.WriteString(a.Caller)
			}
			messageSb.WriteRune(' ')
		}

		messageSb.WriteString(a.Message)

		a.KeyValues = lo.Reject(a.KeyValues, func(kv keyValue, _ int) bool {
			return kv.Key == "callerfunc"
		})
		if len(a.KeyValues) > 0 {
			messageSb.WriteString(": ")
		}
		for idx, item := range a.KeyValues {
			messageSb.WriteString(item.Key + " \"")
			if colorize {
				messageSb.WriteString(color.CyanString(item.Value))
			} else {
				messageSb.WriteString(item.Value)
			}
			messageSb.WriteRune('"')
			if idx < len(a.KeyValues)-1 {
				messageSb.WriteString(", ")
			}
		}
		messageSb.WriteRune('\n')

		return fmt.Fprint(w, messageSb.String())
	}
}