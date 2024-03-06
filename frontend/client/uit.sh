APRILSH_APSHD_PATH=~/.local/bin/apshd 
go test -cover . -args -test.gocoverdir=./coverage/unit
go tool covdata textfmt -i=./coverage/unit -o coverage/profile
go tool cover -html coverage/profile

go tool covdata percent -i=./coverage/unit,./coverage/int 
go tool covdata textfmt -i=./coverage/unit,./coverage/int -o coverage/profile
go tool cover -func coverage/profile
go tool cover -html coverage/profile
