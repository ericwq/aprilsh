// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/exp/slog"
)

// var Log *slog.Logger
// var programLevel = new(slog.LevelVar) // Info by default

type logger struct {
	*slog.Logger
	programLevel *slog.LevelVar
}

var Log *logger

func init() {
	Log = new(logger)
	Log.programLevel = new(slog.LevelVar)

	// start with syslog UDP 514
	writer, err := net.Dial("udp", "localhost:514")
	if err != nil {
		fmt.Println(err)
	}

	if writer != nil {
		Log.Logger = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: Log.programLevel}))
	} else {
		// fallback to stderr
		Log.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: Log.programLevel}))
	}

	slog.SetDefault(Log.Logger)
}

func (l *logger) SetLevel(v slog.Level) {
	l.programLevel.Set(v)
}
