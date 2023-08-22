# Aprilsh

Reborn [mosh](https://mosh.org/) with go.

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

## Status

- 2022/Jul/31: finish the terminal emulator:
  - add scroll buffer support,
  - add color palette support,
  - refine UTF-8 support.
- 2022/Aug/04: start the prediction engine.
- 2022/Aug/29: finish the prediction engine.
  - refine UTF-8 support.
- 2022/Sep/20: finish the UDP network.
- 2022/Sep/28: finish user input state.
- 2022/Sep/29: refine cell width.
- 2022/Oct/02: add terminfo module.
- 2022/Oct/13: finish the Framebuffer for completeness.
- 2022/Oct/14: finish Complete state.
- 2022/Nov/04: finish Display.
- 2022/Nov/08: finish Complete testing.
- 2022/Nov/27: finish Transport and TransportSender.
- 2022/Dec/28: finish command-line parameter parsing and locale validation.
- 2023/Mar/24: solve the locale problem in alpine.
- 2023/Apr/07: support concurrent UDP server.
- 2023/Apr/21: finish server start/stop part.
- 2023/May/01: study [s6](https://skarnet.org/software/s6/) as PID 1 process: [utmps](https://skarnet.org/software/utmps/) require s6, aprilsh also require s6 or similar alternative.
- 2023/May/16: finish [alpine container with openrc support](https://github.com/ericwq/s6)
- 2023/May/30: finish [eric/goutmp](https://github.com/ericwq/goutmp)
- 2023/Jun/07: upgrade to `ericwq/goutmp` v0.2.0.
- 2023/Jun/15: finish `warnUnattached()` part.
- 2023/Jun/21: finish serve() function.
- 2023/Jun/25: re-structure cmd directory.
- 2023/Jul/12: prepare client and server. fix bug in overlay.
- 2023/Jul/19: refine frontend, terminal, util package for test coverage.
- 2023/Jul/24: refine network package for test coverage.
- 2023/Aug/01: start integration test for client.
- 2023/Aug/07: add util.Log and rewrite log related part for other packages.
- 2023/Aug/14: accomplish `exit` command in running aprilsh client.
- 2023/Aug/22: add OSC 112, DECSCUR, XTWINOPS 22,23 support; study CSI u.

## build dependency

```sh
% apk add musl-locales-lang musl-locales utmps-dev
```

Use the above command to add musl locales support and utmps support for alpine. Note alpine only support UTF-8 charmap.

## Architecture view

![aprilsh.svg](img/aprilsh.svg)

- The green part is provided by the system/terminal emulator. Such as [alacritty](https://alacritty.org/) or [kitty](https://sw.kovidgoyal.net/kitty/).
- The cyan part is provided by `Aprilsh`.
- The yellow part is our target terminal application. In the above diagram it's `neovim`.
- Actually the yellow part can be any terminal based application: [emcas](https://www.gnu.org/software/emacs/), [neovim](https://neovim.io/), [htop](https://htop.dev/), etc.
- The rest part is provided by the system.

## Reference

- `mosh` source code analysis [client](https://github.com/ericwq/examples/blob/main/tty/client.md), [server](https://github.com/ericwq/examples/blob/main/tty/server.md)
- [Unicode 14.0 Character Code Charts](http://www.unicode.org/charts/)
- [XTerm Control Sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
- [wezterm Escape Sequences](https://wezfurlong.org/wezterm/escape-sequences.html)
- [Linux man pages](https://linux.die.net/man/)
- [C++ Reference](http://www.cplusplus.com/reference/)
- [Linux logging guide: Understanding syslog, rsyslog, syslog-ng, journald](https://ikshitij.com/linux-logging-guide)
- [CSI u](https://iterm2.com/documentation-csiu.html)
