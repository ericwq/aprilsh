#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

# 
# prepare coverage directory
rm -rf coverage
mkdir -p coverage/unit 
mkdir -p coverage/int

#
# selecting package to cover
PKGS="github.com/ericwq/aprilsh/frontend/server"

#
# build server for test
go build -cover -coverpkg=$PKGS -o ~/.local/bin/apshd .
# go build -race -cover -coverpkg=$PKGS -o ~/.local/bin/apshd .
