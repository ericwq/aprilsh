#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

# 
# prepare coverage directory
rm -rf coverage
mkdir -p coverage/unit 
mkdir -p coverage/int

BuildVersion=`git for-each-ref --count=1 --sort=-taggerdate --format '%(tag)' refs/tags`

echo "Build Start: "$(date "+%F %T.")

#
# selecting package to cover
PKGS="github.com/ericwq/aprilsh/frontend/server"

# 获取 go.mod 项目名,用来指定注入变量位置及输出可以执行程序名称
ModuleName=`head ../../go.mod | grep "^module" | awk '{print $2}'`
# 获取构建时间
BuildTime=$(date "+%F %T")
# 获取构建时 Go 环境信息
GoVersion=`go version`
# 获取构建时 Commit ID
GitCommit=`git rev-parse HEAD`
# 获取构建时的 Git 分支
GitBranch=`git rev-parse --abbrev-ref HEAD`

#
# build server for test
go build -cover -coverpkg=$PKGS -ldflags="-s -w
      -X '${ModuleName}/frontend.BuildVersion=${BuildVersion}'
      -X '${ModuleName}/frontend.GoVersion=${GoVersion}'
      -X '${ModuleName}/frontend.GitCommit=${GitCommit}'
      -X '${ModuleName}/frontend.GitBranch=${GitBranch}'
      -X '${ModuleName}/frontend.BuildTime=${BuildTime}'" -o ~/.local/bin/apshd .
# go build -race -cover -coverpkg=$PKGS -o ~/.local/bin/apshd .
echo "Build End  : "$(date "+%F %T.")
echo "Output to  : GOCOVERDIR=./coverage/int ~/.local/bin/apshd -verbose 1 2>> /tmp/apshd.log"
