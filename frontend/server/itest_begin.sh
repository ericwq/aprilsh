#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

#
# start the server
# here 514 doesn't work, it looks like a bug for coverage. fix it with unit test.
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
#go tool covdata textfmt -i=./coverage/unit,./coverage/int -o coverage/profile

#
# View total coverage
#go tool cover -func coverage/profile
# go tool cover -html coverage/profile
