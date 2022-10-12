# Aprilsh

Reborn [mosh](https://mosh.org/) with go.

## Motivation

[openSSH](https://www.openssh.com/) is excellent. While `mosh` provides better keystroke prediction/latency and is capable of handle WiFi/cellular mobile network roaming problem. But `mosh` is not active anymore and no release [sine 2017](https://github.com/mobile-shell/mosh/issues/1115). Such a good project like `mosh` should keeps developing.

After read through `mosh` source code, I decide to use go to rewrite it. Go is my first choice because the C++ syntax is too complex for me. Go also has excellent support for UTF-8. And remote shell is our daily tools, if it's broken we need a quick fix. The go compiler is fast.

There are several rules for this project:

- Keep the base design of `mosh`: `SSP`, UDP, keystroke prediction.
- Use 3rd party library as less as possible to keep it clean.

There are some goals for this project:

- UTF-8 support, including [emoji and flag](https://unicode.org/emoji/charts/emoji-list.html) support.
- Solve the terminal 24bit color problem.
- Upgrade to [proto3](https://developers.google.com/protocol-buffers/docs/proto3)
- Prove that go is capable of programming terminal application.

The project name `Aprilsh` is derived from `April+sh`. I started this project in April shanghai, it's a remote shell.

## Status

- 2022/Jul/31: finish the terminal emulator:
  - add scroll buffer support,
  - add color palette support,
  - refine UTF-8 support.
- 2022/Aug/04: working on the prediction engine.
- 2022/Aug/29: finish the prediction engine.
  - refine UTF-8 support.
- 2022/Sep/20: finish the UDP network.
- 2022/Sep/28: finish user input state.
- 2022/Sep/29: refine cell width.
- 2022/Oct/02: add terminfo module.
- 2022/Oct/02: working on the Framebuffer for completeness.

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
