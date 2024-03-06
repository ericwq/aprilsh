#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

# 
# prepare coverage directory
rm -rf coverage
mkdir -p coverage/unit 
mkdir -p coverage/int

echo "build server start: $(date '+%F %T')"
#
# selecting package to cover
PKGS="github.com/ericwq/aprilsh/frontend/server"

# prepare for ldflags
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
      -X '${_module_name}/frontend.BuildTime=${_build_time}'" -o ~/.local/bin/apshd .
# go build -race -cover -coverpkg=$PKGS -o ~/.local/bin/apshd .
echo "build server end  : $(date '+%F %T')"
echo "output server to  : ~/.local/bin/apshd"
echo "copy server to    : /usr/bin/apshd"
echo "run with          : GOCOVERDIR=./coverage/int apshd -verbose 2> /tmp/apshd.log"
sudo cp ~/.local/bin/apshd /usr/bin/apshd
echo "export APRILSH_APSHD_PATH=~/.local/bin/apshd"
