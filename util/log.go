// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"io"
	"net"
	"os"

	"golang.org/x/exp/slog"
)

// var Log *slog.Logger
// var programLevel = new(slog.LevelVar) // Info by default

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
	Log.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: Log.programLevel}))
	slog.SetDefault(Log.Logger)
	Log.defaultLogger = slog.Default()
}

func (l *logger) SetLevel(v slog.Level) {
	l.programLevel.Set(v)
}

// network: udp, address: localhost:514. check net.Dial() for detail
func (l *logger) SetupSyslog(network string, address string) {
	writer, err := net.Dial(network, address)
	if writer != nil {
		l.Logger = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: Log.programLevel}))
		slog.SetDefault(Log.Logger)
		l.defaultLogger = slog.Default()
	} else {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (l *logger) SetOutput(w io.Writer) {
	l.Logger = slog.New(slog.NewTextHandler(w, nil))
}

func (l *logger) Restore() {
	l.Logger = l.defaultLogger
}
