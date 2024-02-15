// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import "fmt"

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
	GitTag    string // build version
	GoVersion string // Go version
	BuildTime string // build time
	GitCommit string // git commit id
	GitBranch string // git branch name
)

func PrintVersion() {
	fmt.Printf("git tag   \t: %s\n", GitTag)
	fmt.Printf("git commit\t: %s\n", GitCommit)
	fmt.Printf("git branch\t: %s\n", GitBranch)
	fmt.Printf("go version\t: %s\n", GoVersion)
	fmt.Printf("build time\t: %s\n\n", BuildTime)
	fmt.Printf(VersionInfo)
}
