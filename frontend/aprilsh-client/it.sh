#!/bin/sh

#
# https://go.dev/testing/coverage/#running
# https://go.dev/blog/integration-test-coverage
# https://dustinspecker.com/posts/go-combined-unit-integration-code-coverage/
#
BUILDARGS="$*"
#
# Terminate the test if any command below does not complete successfully.
#
set -e
#
# Build server binary for testing purposes.
#
cd ~/develop/aprilsh/frontend/aprilsh-server/ 
go build -cover -o ~/develop/aprilsh/frontend/aprilsh-client/server .
#
# Build client binary for testing purposes.
#
cd ../aprilsh-client/ 
go build -cover -o client client.go
#
# Setup
#
rm -rf covdata
mkdir covdata
#
# start the server
#
GOCOVERDIR=covdata ./server &
server_id =`ps -o pid,user,comm | grep ide | grep -v 'server' | grep server`

# 
# start client
#
GOCOVERDIR=covdata ./client 
client_id =`ps -o pid,user,comm | grep ide | grep -v 'client' | grep server`

# 
# clean the server and client
#
kill -9 $server_id
kill -9 $client_id

# 
# Reporting percent statements covered
#
go tool covdata percent -i=covdata

# 
# Converting to legacy text format
#
go tool covdata textfmt -i=covdata -o profile.txt
