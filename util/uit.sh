rm -rf coverage
mkdir -p coverage/unit
go test -cover . -args -test.gocoverdir=./coverage/unit
go tool covdata textfmt -i=./coverage/unit -o coverage/profile
go tool cover -html coverage/profile
