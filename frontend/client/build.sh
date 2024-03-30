#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

# 
# prepare coverage directory
rm -rf coverage
mkdir -p coverage/unit 
mkdir -p coverage/int

# selecting package for integration coverage test
PKGS="github.com/ericwq/aprilsh/frontend/client"

# prepare for ldflags
_module_name=$(head ../../go.mod | grep '^module' | awk '{print $2}')
_go_version=$(go version | grep 'version' | awk '{print $3}')
_git_tag=$(git describe --tags)
_git_commit=$(git rev-parse --short HEAD)
_git_branch=$(git rev-parse --abbrev-ref HEAD)

# set ldflags
ldflags="-s -w \
	-X $_module_name/frontend.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%S) \
	-X $_module_name/frontend.GitBranch=$_git_branch \
	-X $_module_name/frontend.GitCommit=$_git_commit \
	-X $_module_name/frontend.GitTag=$_git_tag \
	-X $_module_name/frontend.GoVersion=$_go_version \
	"
# required for github.com/docker/docker
export GO111MODULE=auto

# build client for test
echo "build client start: $(date '+%F %T')"
go build -cover -coverpkg=$PKGS -ldflags="$ldflags" -o ~/.local/bin/apsh
echo "build client end  : $(date '+%F %T')"
echo "output client to  : ~/.local/bin/apsh"
echo "run with          : GOCOVERDIR=./coverage/int  ~/.local/bin/apsh -verbose ide@localhost"

# chech version number
# if [ "$1" != "" ]; then Version=$1;else
#     read -p "Input Build Version: " Version; if [ "$Version" = "" ]; then echo "The input cannot be empty";exit;fi
# fi
