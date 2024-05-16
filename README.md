Aprilsh: remote shell support intermittent or mobile network. inspired by [mosh](https://mosh.org/) and [zutty](https://github.com/tomscii/zutty). aprilsh is a remote shell based on UDP, authenticate user via openssh.

## Install

#### reqirement
- [open-ssh](https://www.openssh.com/) is a must reqirement, openssh is needed to perform user authentication.
- [locale support](https://git.adelielinux.org/adelie/musl-locales/-/wikis/home) is a must reqirement.
- [ncurses and terminfo](https://invisible-island.net/ncurses/) is a must requirement.
- [systmd](https://systemd.io/) reuired for redhat linux family (fedora, centos, redhat).
- [utmps](https://skarnet.org/software/utmps/) reuired only for alpine
- [openrc](https://github.com/OpenRC/openrc) required only for alpine.
- [logrotate](https://github.com/logrotate/logrotate) reuired only for alpine.

if you perfer to build aprilsh manually, please refer to [this document](doc/install-alpine.md)

#### Alpine linux
Before start apshd, you need to make sure you can ssh login to the target server, please refer to [this doc](doc/ssh-alpine.md) to setup a ssh enabled docker container.

Note: aprilsh is still waiting for aports approval. For now please use the following private repository. The private repository only provide `x86_64` packages. Refer to [build doc](doc/build.md) to know how to build apk packages and private repositories.
```sh
wget -P /etc/apk/keys/ https://ericwq.github.io/alpine/packager-663ebf9b.rsa.pub    # add public key
echo "https://ericwq.github.io/alpine/v3.19/testing" >> /etc/apk/repositories       # add private repository
apk update                                                                          # update repositories metadata
apk add aprilsh                                                                     # install client and server
```
Now you can ssh login to the server and the aprilsh is installed, it's time to start apshd server and login with apsh.
```sh
rc-service apshd start          # start apshd server
apsh -m 100 eric@localhost:8022 # apsh login to server
```
Note: when aports finally approve aprilsh, the above private repository will be replaced by official testing repositories. The testing repositories will provide all architecture packages.
```sh
echo "https://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories  # add testing repositories
```

#### Fedora, CentOS, Redhat linux
Note: This is a private yum/dnf repositories, it only provides `x86_64` packages. Refer to [rpms doc](https://codeberg.org/ericwq/rpms#build-rpm-packages) to understand how to build rpm packags and dnf repositories.
```sh
rpm --import https://ericwq.codeberg.page/RPM-GPG-KEY-wangqi            # import public key to rpm DB
dnf config-manager --add-repo https://ericwq.codeberg.page/aprilsh.repo # add new repo to dnf repository
dnf install aprilsh                                                     # install client and server
```
Before start apshd, you need to make sure you can ssh login to the target server, please refer to [this doc](doc/ssh-fedora.md) to setup a ssh enabled docker container.

Now you can ssh login to the server, it's time to start apshd service and login with apsh.
```sh
sudo systemctl start apshd.service      #start apshd service
sudo journalctl -f -u apshd.service     #keep reading the latest apshd.service log
apsh user@host                          # start apsh client on different host
```
#### MacOS
```sh
brew tap ericwq/utils       # add tap to homebrew
brew install aprilsh        # only install aprilsh client
```
Refer to [homebrew doc](https://github.com/ericwq/homebrew-utils) to know how to create homebrew package and tap.

## Motivation

[openSSH](https://www.openssh.com/) is excellent. While `mosh` provides better keystroke prediction/latency and is capable of handle WiFi/cellular mobile network roaming. But `mosh` project is not active anymore and no release [sine 2017](https://github.com/mobile-shell/mosh/issues/1115). Such a good project like `mosh` should keeps developing.

After read through `mosh` source code, I decide to rewrite it with golang. Go is my first choice because the C++ syntax is too complex. Go also has excellent support for UTF-8 and multithreaded programming. The last reason: go compiler is faster than c++ compiler.

There are several rules for this project:

- Keep the base design of `mosh`: `SSP`, UDP, keystroke prediction.
- Use 3rd party library as less as possible to keep it clean.

There are also some goals for this project:

- Full UTF-8 support, including [emoji and flag](https://unicode.org/emoji/charts/emoji-list.html) support.
- Support the terminal 24bit color.
- Upgrade to [proto3](https://developers.google.com/protocol-buffers/docs/proto3)
- Use terminfo database for better compatibility.
- Prove golang is a good choice for terminal developing.

The project name `Aprilsh` is derived from `April+sh`. This project started in shanghai April 2022, and it's a remote shell.
Use the above command to add musl locales support and utmps support for alpine. Note alpine only support UTF-8 charmap.

## Architecture view

![aprilsh.svg](img/aprilsh.svg)

- The green part is provided by the system/terminal emulator. Such as [alacritty](https://alacritty.org/) or [kitty](https://sw.kovidgoyal.net/kitty/).
- The cyan part is provided by `Aprilsh`.
- The yellow part is our target terminal application. In the above diagram it's `neovim`.
- Actually the yellow part can be any terminal based application: [emcas](https://www.gnu.org/software/emacs/), [neovim](https://neovim.io/), [htop](https://htop.dev/), etc.
- The rest part is provided by the system.

## Changelog

Ready for early acess. The missing part is prediction engine tuning. Check [here](doc/changelog.md) for history deatil.

## Reference

- `mosh` source code analysis [client](https://github.com/ericwq/examples/blob/main/tty/client.md), [server](https://github.com/ericwq/examples/blob/main/tty/server.md)
- [Unicode 14.0 Character Code Charts](http://www.unicode.org/charts/)
- [XTerm Control Sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
- [wezterm Escape Sequences](https://wezfurlong.org/wezterm/escape-sequences.html)
- [Linux man pages](https://linux.die.net/man/)
- [C++ Reference](http://www.cplusplus.com/reference/)
- [Linux logging guide: Understanding syslog, rsyslog, syslog-ng, journald](https://ikshitij.com/linux-logging-guide)
- [Benchmarking in Golang: Improving function performance](https://blog.logrocket.com/benchmarking-golang-improve-function-performance/)

Need some time to figure out how to support CSI u in aprilsh.

- [Comprehensive keyboard handling in terminals](https://sw.kovidgoyal.net/kitty/keyboard-protocol/#functional-key-definitions)
- [feat(tui): query terminal for CSI u support](https://github.com/neovim/neovim/pull/18181)
- [Fix Keyboard Input on Terminals - Please](https://www.leonerd.org.uk/hacks/fixterms/)
- [xterm modified-keys](https://invisible-island.net/xterm/modified-keys.html)
