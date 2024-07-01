
### add dependencies

build dependency.
```sh
apk add go protoc utmps-dev ncurses musl-locales ncurses-terminfo protoc-gen-go
```

run dependency.
```sh
apk add musl-locales utmps ncurses logrotate ncurses-terminfo openssh-server

```
### build from source
```sh
git clone https://github.com/ericwq/aprilsh.git
cd aprilsh/frontend/server
./build.sh
cd aprilsh/frontend/client
./build.sh
cd aprilsh
APRILSH_APSHD_PATH="/home/ide/.local/bin/apshd" \
go test -tags=utmps $(go list ./... | grep -Ev '(data|protobufs)')

```
### install for alpine

add testing repositories to your alpine system, you need the root privilege to do that.
```sh
echo "https://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories
apk update
```
test souce code:
```sh
APRILSH_APSHD_PATH="/home/ide/.local/bin/apshd" \
go test -tags=utmps $(go list ./... | grep -Ev '(data|protobufs)')
```
add aprilsh, which includes aprilsh-server, aprilsh-client, aprilsh-openrc:
```sh
apk add aprilsh
```

run apshd (aprilsh server) as openrc service.
```sh
rc-service apshd start
```

or run apshd (aprilsh server) manually.
```sh
apshd 2>/var/log/apshd &
```
by default apshd listen on udp localhost:8100.
```txt
openrc-nvide:~# netstat -lup
Active Internet connections (only servers)
Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program name
udp        0      0 localhost:8100          0.0.0.0:*                           45561/apshd
openrc-nvide:~#
```
now login to the system with apsh (aprilsh client), note the `motd`(welcome message) depends on you alpine system config.
```txt
qiwang@Qi15Pro client % apsh ide@localhost
openrc-nvide:0.10.2

Lua, C/C++ and Golang Integrated Development Environment.
Powered by neovim, luals, gopls and clangd.
ide@openrc-nvide:~ $
```
if you login on two terminals, on the server, there will be two server processes serve the clients. the following shows `apshd` serve two clients. one is`:8101`, the other is ':8102'
```sh
openrc-nvide:~# netstat -lup
Active Internet connections (only servers)
Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program name
udp        0      0 localhost:8100          0.0.0.0:*                           45561/apshd
udp        0      0 :::8101                 :::*                                45647/apshd
udp        0      0 :::8102                 :::*                                45612/apshd
openrc-nvide:~#
```
enjoy the early access for aprilsh.
