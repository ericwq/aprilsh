#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

# prepare for ldflags
_module_name=$(head go.mod | grep '^module' | awk '{print $2}')
_go_version=$(go version | grep 'version' | awk '{print $3}')
_git_tag=$(git describe --tags)
_git_commit=$(git rev-parse --short HEAD)
_git_branch=$(git rev-parse --abbrev-ref HEAD)

# prepre build tags
_osType=$(uname)
if [ "${_osType}" == 'Darwin' ]; then
  _musl=$(otool -L /bin/ls | grep 'musl' | head -n 1 | awk '{print $1}')
else
  _musl=$(ldd /bin/ls | grep 'musl' | head -n 1 | awk '{print $1}')
fi

if [ "$_musl" == "" ]; then
  _build_tag="-tags utmp"
else
  _build_tag="-tags utmps"
fi
echo "build with tags   : $_build_tag"

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

_dst="$HOME/.local/bin"
# build server and client
if [ "${_osType}" == 'Linux' ]; then
  go build $_build_tag -ldflags="$ldflags" -o $_dst/apshd ./frontend/server
  echo "build apshd to    : $_dst"
fi
go build -ldflags="$ldflags" -o $_dst/apsh ./frontend/client
echo "build apsh  to    : $_dst"
# echo "run with          : $_dst/apsh -vv ide@localhost 2>>/tmp/apsh01.log"

# run test
if [ "${_osType}" == 'Linux' ]; then
  echo "run test          :"
  APRILSH_APSHD_PATH="$_dst/apshd" \
    go test $_build_tag $(go list ./... | grep -Ev '(data|protobufs)')
fi
