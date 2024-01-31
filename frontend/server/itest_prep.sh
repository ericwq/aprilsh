#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

# 
# prepare coverage directory
rm -rf coverage
mkdir -p coverage/unit 
mkdir -p coverage/int

# BuildVersion=`git for-each-ref --count=1 --sort=-taggerdate --format '%(tag)' refs/tags`
BuildVersion=`git describe --tags`
echo "build server start: `date '+%F %T'`"
#
# selecting package to cover
PKGS="github.com/ericwq/aprilsh/frontend/server"
# get go module name
ModuleName=`head ../../go.mod | grep "^module" | awk '{print $2}'`
# get build time
BuildTime=`date '+%F %T'`
# get go version
GoVersion=`go version | grep "version" | awk '{print $3,$4}'`
# get git commit ID
GitCommit=`git rev-parse HEAD`
# get git branch
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
echo "build server end  : `date '+%F %T'`"
echo "output server to  : ~/.local/bin/apshd"
echo "move server to    : /usr/bin/apshd"
echo "run with          : GOCOVERDIR=./coverage/int apshd -verbose 1 2>> /tmp/apshd.log"
sudo mv ~/.local/bin/apshd /usr/bin/apshd
