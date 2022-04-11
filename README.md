# Aprilsh

a [mosh](https://mosh.org/) like tools, golang implementation.

## Motivation

[openSSH](https://www.openssh.com/) is excellent. While [mosh](https://mosh.org/) provides better keystroke prediction and mosh is capable of handle WiFi/3G/4G/5G roaming problem. But mosh is not active anymore and no release [sine 2017](https://github.com/mobile-shell/mosh/issues/1115). Such a good project like mosh should keeps developing.

After read through mosh source code, I decide to use golang to rewrite it. Go is my first choice because the C++ syntax is too complex for me. There is several rules for this project.

- Keep the base design of mosh.
- Use as much as less 3rd party library to keep the implementation clean.

There is some goals:

- Solve the terminal 24bit color support problem.
- Verify Golang is capable of programming terminal application.

The project name `Aprilsh` is derived from `April+sh`. We started this project in April, it's a remote shell.

## Reference

- mosh source code analysis [client](https://github.com/ericwq/examples/blob/main/tty/client.md), [server](https://github.com/ericwq/examples/blob/main/tty/server.md)
- [Unicode 14.0 Character Code Charts](http://www.unicode.org/charts/)
- [XTerm Control Sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
- [Linux man pages](https://linux.die.net/man/)
- [C++ Reference](http://www.cplusplus.com/reference/)
