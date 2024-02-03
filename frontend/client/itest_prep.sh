#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

# 
# prepare coverage directory
rm -rf coverage
mkdir -p coverage/unit 
mkdir -p coverage/int

# chech version number
# if [ "$1" != "" ]; then Version=$1;else
#     read -p "Input Build Version: " Version; if [ "$Version" = "" ]; then echo "The input cannot be empty";exit;fi
# fi
GitTag=`git describe --tags`
echo "build client start: `date '+%F %T'`"
#
# selecting package to cover
PKGS="github.com/ericwq/aprilsh/frontend/client"

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
      -X '${ModuleName}/frontend.GitTag=${GitTag}'
      -X '${ModuleName}/frontend.GoVersion=${GoVersion}'
      -X '${ModuleName}/frontend.GitCommit=${GitCommit}'
      -X '${ModuleName}/frontend.GitBranch=${GitBranch}'
      -X '${ModuleName}/frontend.BuildTime=${BuildTime}'" -o ~/.local/bin/apsh .
# go build -race -cover -coverpkg=$PKGS -o ~/.local/bin/apshd .
echo "build client end  : `date '+%F %T'`"
echo "output client to  : ~/.local/bin/apsh"
echo "run with          : GOCOVERDIR=./coverage/int  ~/.local/bin/apsh -verbose 1 -pwd password ide@localhost 2>> /tmp/apsh00.log"
