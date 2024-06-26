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
PKGS="github.com/ericwq/aprilsh/frontend/server"

# prepare for ldflags
_module_name=$(head ../../go.mod | grep '^module' | awk '{print $2}')
_go_version=$(go version | grep 'version' | awk '{print $3}')
_git_tag=$(git describe --tags)
_git_commit=$(git rev-parse --short HEAD)
_git_branch=$(git rev-parse --abbrev-ref HEAD)

_osType=$(uname)
if [ "${_osType}" == 'Darwin' ]
then
_musl=$(otool -L /bin/ls |  grep 'musl'|head -n 1| awk '{print $1}')
else
_musl=$(ldd /bin/ls | grep 'musl'|head -n 1| awk '{print $1}')
fi

if [ "$_musl" == "" ]
then
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

# build server for test
echo "build server start: $(date '+%F %T')"
go build $_build_tag -cover -coverpkg=$PKGS -ldflags="$ldflags" -o ~/.local/bin/apshd
echo "build server end  : $(date '+%F %T')"
echo "output server to  : ~/.local/bin/apshd"

# for linux sudo copy apshed to /usr/bin
if [ "${_osType}" == 'Linux' ]
then
  echo "copy server to    : /usr/bin/apshd"
  sudo cp ~/.local/bin/apshd /usr/bin/apshd
fi

echo "run with          : GOCOVERDIR=./coverage/int apshd -verbose 2> /tmp/apshd.log"
