# Aprilsh

Reborn [mosh](https://mosh.org/) with go.

## Motivation

[openSSH](https://www.openssh.com/) is excellent. While `mosh` provides better keystroke prediction/latency and is capable of handle WiFi/cellular mobile network roaming problem. But `mosh` is not active anymore and no release [sine 2017](https://github.com/mobile-shell/mosh/issues/1115). Such a good project like `mosh` should keeps developing.

After read through `mosh` source code, I decide to use go to rewrite it. Go is my first choice because the C++ syntax is too complex for me. Go also has excellent support for UTF-8 and goroutine. And remote shell is my daily tools, if it's broken I need a quick fix. The last reason is go compiler is faster than c++.

There are several rules for this project:

- Keep the base design of `mosh`: `SSP`, UDP, keystroke prediction.
- Use 3rd party library as less as possible to keep it clean.

There are also some goals for this project:

- Full UTF-8 support, including [emoji and flag](https://unicode.org/emoji/charts/emoji-list.html) support.
- Solve the terminal 24bit color problem.
- Upgrade to [proto3](https://developers.google.com/protocol-buffers/docs/proto3)
- Use terminfo database for better compatibility.
- Prove that go is capable of programming terminal application.

The project name `Aprilsh` is derived from `April+sh`. I started this project in April shanghai, it's a remote shell.

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

## musl locales

```sh
% apk add musl-locales-lang musl-locales
```

Use the above command to add musl locales support for alpine. Only UTF-8 charmap is supported

## glibc locales

see [here](http://blog.fpliu.com/it/software/GNU/glibc#alpine) and [there](https://zhuanlan.zhihu.com/p/151852282) for install glibc in alpine linux.

```sh
% curl -L -o /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub
% export APK_GLIBC_VERSION=2.35-r0
% export APK_GLIBC_BASE_URL="https://github.com/sgerrand/alpine-pkg-glibc/releases/download/${APK_GLIBC_VERSION}"
% curl -LO "${APK_GLIBC_BASE_URL}/glibc-${APK_GLIBC_VERSION}.apk"
% curl -LO "${APK_GLIBC_BASE_URL}/glibc-bin-${APK_GLIBC_VERSION}.apk"
% curl -LO "${APK_GLIBC_BASE_URL}/glibc-dev-${APK_GLIBC_VERSION}.apk"
% curl -LO "${APK_GLIBC_BASE_URL}/glibc-i18n-${APK_GLIBC_VERSION}.apk"
% apk add glibc-${APK_GLIBC_VERSION}.apk glibc-bin-${APK_GLIBC_VERSION}.apk glibc-dev-${APK_GLIBC_VERSION}.apk glibc-i18n-${APK_GLIBC_VERSION}.apk
% rm glibc-*
% export PATH=/usr/glibc-compat/bin:$PATH
```

Intall required locale.

```sh
% localedef -i zh_CN -f GB18030 zh_CN.GB18030
% localedef -i en_US -f UTF-8 en_US.UTF-8
```

check [here](https://gist.github.com/larzza/0f070a1b61c1d6a699653c9a792294be) for install glibc in alpine docker image.

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
