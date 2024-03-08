APRILSH_APSHD_PATH=~/.local/bin/apshd 
go test -cover . -args -test.gocoverdir=./coverage/unit
go tool covdata textfmt -i=./coverage/unit -o coverage/profile
go tool cover -html coverage/profile

go tool covdata percent -i=./coverage/unit,./coverage/int 
go tool covdata textfmt -i=./coverage/unit,./coverage/int -o coverage/profile
go tool cover -func coverage/profile
go tool cover -html coverage/profile

# 1. login, change terminal size, log out.
#	 GOCOVERDIR=./coverage/int  ~/.local/bin/apsh ide@localhost
# 2. login with plublic key
#	 GOCOVERDIR=./coverage/int  ~/.local/bin/apsh -i ~/.ssh/id_ed25519 ide@localhost
#	 1. input correct password: faild login
#	 2. input incorrect password: login with ssh agent
# 3. login with remove ~/.ssh/know_hosts
#	 rm ~/.ssh/known_hosts*
#	 GOCOVERDIR=./coverage/int  ~/.local/bin/apsh ide@localhost
#	 1. reply no
#	 2. reply enter
#	 3. reply yes
# 4. login with wrong host name or wrong user name
# 	 GOCOVERDIR=./coverage/int  ~/.local/bin/apsh -verbose ide@localhost2
# 	 GOCOVERDIR=./coverage/int  ~/.local/bin/apsh -verbose ide2@localhost
# 5. close ssh agent, login with wrong plublic key, thus use password auth
#	 export SSH_AUTH_SOCK=
#	 GOCOVERDIR=./coverage/int  ~/.local/bin/apsh -i ~/.ssh/id_777 ide@localhost
# 6. login then kill client process
#	 GOCOVERDIR=./coverage/int  ~/.local/bin/apsh -verbose ide@localhost
#	 kill <pid>
