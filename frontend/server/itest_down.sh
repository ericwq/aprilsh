#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

#
# start the server
# - output debug level log information to specified file
GOCOVERDIR=./coverage/int ~/.local/bin/aprilsh-server -verbose 1 2>> /tmp/aprilsh-server.log &
spid=$!

#
# start client with the following command on remote machine
# ./client -verbose 1 -pwd password ide@172.17.0.3 2>> /tmp/aprilsh.log
# check the client is automaticlly finished when the server is down

#
# kill the server
sleep 5
kill $spid
