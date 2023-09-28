#!/bin/sh

#
# Terminate the test if any command below does not complete successfully.
set -e

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

