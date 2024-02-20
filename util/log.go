// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"io"
	"os"

	"log/slog"
)

const (
	LevelTrace = slog.Level(-8)
	DebugLevel = 1
	TraceLevel = 2
)

type logger struct {
	*slog.Logger
	defaultLogger *slog.Logger
	programLevel  *slog.LevelVar
}

var Log *logger

func init() {
	// default logger write to stderr
	Log = new(logger)
	Log.programLevel = new(slog.LevelVar)
	Log.SetLevel(slog.LevelInfo)
	Log.SetOutput(os.Stderr)
}

func (l *logger) SetLevel(v slog.Level) {
	l.programLevel.Set(v)
}

func (l *logger) addSource() bool {
	if l.programLevel.Level() <= slog.LevelDebug {
		return true
	}
	return false
}

func (l *logger) SetOutput(w io.Writer) {
	ho := &slog.HandlerOptions{AddSource: Log.addSource(), Level: Log.programLevel}
	l.Logger = slog.New(slog.NewTextHandler(w, ho))
	slog.SetDefault(Log.Logger)
	l.defaultLogger = slog.Default()
}

func (l *logger) Restore() {
	l.Logger = l.defaultLogger
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
