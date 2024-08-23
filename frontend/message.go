// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const (
	AprilshMsgOpen     = "open aprilsh:"
	AprishMsgClose     = "close aprilsh:"
	AprilshPackageName = "aprilsh"
	CommandServerName  = "apshd"
	CommandClientName  = "apsh"
	TimeoutIfNoResp    = 60000
	TimeoutIfNoConnect = 15000

	VersionInfo = `Copyright (c) 2022~2024 wangqi <ericwq057@qq.com>
License MIT: <https://en.wikipedia.org/wiki/MIT_License>.
This is free software: you are free to change and redistribute it.
There is NO WARRANTY, to the extent permitted by law.

remote shell support intermittent or mobile network.
`
)

var (
	GitTag      string // build version
	GoVersion   string // Go version
	BuildTime   string // build time
	GitCommit   string // git commit id
	GitBranch   string // git branch name
	DefaultPort = 8100 // https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers
)

func PrintVersion() {
	fmt.Printf("git tag   \t: %s\n", GitTag)
	fmt.Printf("git commit\t: %s\n", GitCommit)
	fmt.Printf("git branch\t: %s\n", GitBranch)
	fmt.Printf("go version\t: %s\n", GoVersion)
	fmt.Printf("build time\t: %s\n\n", BuildTime)
	fmt.Print(VersionInfo)
}

func PrintUsage(hint string, usage ...string) {
	if hint != "" {
		var header string
		if len(usage) != 0 {
			header = "Hints: "
		}
		fmt.Printf("%s%s\n", header, hint)
	}
	if len(usage) > 0 {
		fmt.Printf("%s", usage[0])
	}
}

func EncodeTerminalCaps(caps map[int]string) []byte {
	jsonData, _ := json.Marshal(caps)
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(jsonData)))
	base64.StdEncoding.Encode(dst, []byte(jsonData))

	return dst
}

func DecodeTerminalCaps(str []byte) (map[int]string, error) {
	caps := make(map[int]string)

	dst := make([]byte, base64.StdEncoding.DecodedLen(len(str)))
	n, err := base64.StdEncoding.Decode(dst, []byte(str))
	if err != nil {
		return nil, err
	}
	dst = dst[:n]

	err = json.Unmarshal(dst, &caps)
	if err != nil {
		return nil, err
	}
	return caps, nil
}
