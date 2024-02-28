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
echo "build client start: $(date '+%F %T')"
#
# selecting package to cover
PKGS="github.com/ericwq/aprilsh/frontend/client"

# get go module name
_module_name=$(head ../../go.mod | grep "^module" | awk '{print $2}')
_build_time=$(date '+%F %T')
_go_version=$(go version | grep "version" | awk '{print $3,$4}')
_git_tag=$(git describe --tags)
_git_commit=$(git rev-parse HEAD)
_git_branch=$(git rev-parse --abbrev-ref HEAD)
#
# build server for test
go build -cover -coverpkg=$PKGS -ldflags="-s -w
      -X '${_module_name}/frontend.GitTag=${_git_tag}'
      -X '${_module_name}/frontend.GoVersion=${_go_version}'
      -X '${_module_name}/frontend.GitCommit=${_git_commit}'
      -X '${_module_name}/frontend.GitBranch=${_git_branch}'
      -X '${_module_name}/frontend.BuildTime=${_build_time}'" -o ~/.local/bin/apsh .
# go build -race -cover -coverpkg=$PKGS -o ~/.local/bin/apshd .
echo "build client end  : $(date '+%F %T')"
echo "output client to  : ~/.local/bin/apsh"
echo "run with          : GOCOVERDIR=./coverage/int  ~/.local/bin/apsh -verbose ide@localhost"
