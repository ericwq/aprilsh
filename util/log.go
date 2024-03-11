// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"context"
	"io"
	"os"

	"log/slog"
)

const (
	LevelTrace = slog.Level(-8)
	LevelFatal = slog.Level(12)
	DebugLevel = 1
	TraceLevel = 2
)

var Logger *myLogger
var levelNames = map[slog.Leveler]string{
	LevelTrace: "TRACE",
	LevelFatal: "FATAL",
}

type myLogger struct {
	*slog.Logger
	addSource bool
	logLevel  *slog.LevelVar
}

func init() {
	// default logger write to stderr
	Logger = new(myLogger)
	Logger.logLevel = new(slog.LevelVar)
	Logger.SetLevel(slog.LevelInfo)
	Logger.AddSource(false)
	Logger.SetOutput(os.Stderr)
}

func (l *myLogger) SetLevel(v slog.Level) {
	l.logLevel.Set(v)
}

func (l *myLogger) AddSource(add bool) {
	Logger.addSource = add
}

// how to replace a line in file,sample
// sed -i 's/.*defer util\.Log\.Restore.*//g' encrypt/encrypt_test.go
//

func (l *myLogger) SetOutput(w io.Writer) {
	ho := &slog.HandlerOptions{
		AddSource: Logger.addSource,
		Level:     Logger.logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				levelLabel, exists := levelNames[level]
				if !exists {
					levelLabel = level.String()
				}

				a.Value = slog.StringValue(levelLabel)
			}

			return a
		},
	}
	l.Logger = slog.New(slog.NewTextHandler(w, ho)).With("pid", os.Getpid())
	slog.SetDefault(l.Logger)
}

func (l *myLogger) CreateLogger(w io.Writer, source bool, level slog.Level) {
	ho := &slog.HandlerOptions{
		AddSource: source,
		Level:     level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				levelLabel, exists := levelNames[level]
				if !exists {
					levelLabel = level.String()
				}

				a.Value = slog.StringValue(levelLabel)
			}

			return a
		},
	}
	l.Logger = slog.New(slog.NewTextHandler(w, ho)).With("pid", os.Getpid())
	// slog.SetDefault(l.Logger)
}

func (l *myLogger) Trace(msg string, args ...any) {
	l.Logger.Log(context.Background(), LevelTrace, msg, args...)
}

// network: udp, address: localhost:514. check net.Dial() for detail
// func (l *logger) SetupSyslog(network string, address string) error {
// 	writer, err := net.Dial(network, address)
// 	// writer, err := syslog.New(syslog.LOG_DAEMON|syslog.LOG_LOCAL7, "aprilsh")
// 	if err != nil {
// 		return err
// 	}
//
// 	ho := &slog.HandlerOptions{AddSource: l.isDebugLevel(), Level: Log.programLevel}
// 	l.Logger = slog.New(slog.NewTextHandler(writer, ho))
// 	slog.SetDefault(Log.Logger)
// 	l.defaultLogger = slog.Default()
// 	return nil
// }

// // create log file based on prefix under tmp directory. such as aprilsh-PID.log
// func (l *logger) CreateLogFile(prefix string) (*os.File, error) {
// 	name := joinPath(os.TempDir(), fmt.Sprintf("%s-%d.%s", prefix, os.Getpid(), "log"))
// 	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return file, nil
// }

// func joinPath(dir, name string) string {
// 	if len(dir) > 0 && os.IsPathSeparator(dir[len(dir)-1]) {
// 		return dir + name
// 	}
// 	return dir + string(os.PathSeparator) + name
// }
