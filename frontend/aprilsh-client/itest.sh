#!/bin/sh

#
# https://go.dev/testing/coverage/#running
# https://go.dev/blog/integration-test-coverage
# https://dustinspecker.com/posts/go-combined-unit-integration-code-coverage/
#
BUILDARGS="$*"
#
# comma seperated package list
#
PKG="github.com/ericwq/aprilsh/frontend/aprilsh-client"
PKGARGS="-coverpkg=$PKG"
#
# Terminate the test if any command below does not complete successfully.
#
set -e
#
# Build server binary for testing purposes.
#
cd ../aprilsh-server/
go build -cover -o ../aprilsh-client/server .
echo "---build server"
#
# Build client binary for testing purposes.
#
cd ../aprilsh-client/ 
go build -cover $PKGARGS -o client .
echo "---build client"
#
# Setup
#
rm -rf coverage
mkdir -p coverage/unit -p coverage/int
#
# Run unit test to collect coverage
#
go test -cover . -args -test.gocoverdir=./coverage/unit
echo "---perform unit test"
#
# start the server
#
GOCOVERDIR=./coverage/int ./server &
server_id=$!
echo "---start server"
# 
# start client
#
GOCOVERDIR=./coverage/int ./client ide@localhost
client_id=$!
sleep 2
echo "---start client"
# 
# clean the server and client
#
echo "kill server[$server_id]"
echo "kill client[$client_id]"
kill $server_id
# kill -9 $client_id

# 
# Retrieve total coverage
#
go tool covdata percent -i=./coverage/unit,./coverage/int -pkg=$PKG

# 
# Converting to legacy text format
#
go tool covdata textfmt -i=./coverage/unit,./coverage/int -o coverage/profile -pkg=$PKG

#
# View total coverage
#
go tool cover -func coverage/profile
