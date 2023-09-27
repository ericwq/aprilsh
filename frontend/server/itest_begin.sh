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
# go build -cover -coverpkg=$PKGS -o ~/.local/bin/aprilsh-server server.go
go build -cover -coverpkg=$PKGS -o ./server .

#
# start the server
GOCOVERDIR=./coverage/int ./server -verbose 514 2>> /tmp/aprilsh-server.log &
spid=$!

#
# begin client connection
GOCOVERDIR=./coverage/int ./server -b -p 8080
GOCOVERDIR=./coverage/int ./server -b

#
# kill the server
kill $spid
kill -9 $spid
# echo "-- kill server $spid"

#
# Run unit tests to collect coverage
go test -cover . -args -test.gocoverdir=./coverage/unit

#
# Retrieve total coverage
# go tool covdata percent -i=./coverage/unit,./coverage/int

#
# Convert total coverage to cover profile
go tool covdata textfmt -i=./coverage/unit,./coverage/int -o coverage/profile

#
# View total coverage
go tool cover -func coverage/profile
# go tool cover -html coverage/profile
