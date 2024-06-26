# Basic design

## Parser design

What we need is a terminal emulator, which can:

- accept and parse the mix of data and escape sequences and perform the action accordingly.
- calculate the difference between states of terminal emulator and output mix.
- apply the output mix to replicate the state of terminal emulator.

In real world, the user may choose different terminal emulator. The `TERM` environment variable should be propagated from client to the server, just like SSH does. Different `TERM` means different terminal capability.

How to adapt to this situation: support different terminal emulator, such as `xterm-256color`,
`rxvt-256color`, `alacritty`, still need a solution.

- First solution: write separate independent parser and actor for different `terminfo` entry. The different implementation share the same terminal emulator data structure. The implementation is chosen by `TERM` at runtime.
- Second solution: write a base class which can parse and act on a base `terminfo` entry, which could be `st-256color`. (We don't choose `xterm-256color` because `xterm` is too complex) The other `terminfo` entry is implemented by a extend class. The different implementation share the same terminal emulator data structure. The implementation is chosen by `TERM` at runtime.
- Third solution: a [state machine](https://vt100.net/emu/dec_ansi_parser). We choose the third solution because it's the classical solution to our problem.

## Terminfo design

Both solutions suggest we can support `terminfo` entries one by one, step by step. The scope of `terminfo` entries can be narrowed down to `ncurses-terminfo-base` package (the alpine linux platform).

```sh
ide@nvide-ssh:/etc/terminfo $ apk info -L ncurses-terminfo-base
ncurses-terminfo-base-6.3_p20220423-r0 contains:
etc/terminfo/a/alacritty
etc/terminfo/a/ansi
etc/terminfo/d/dumb
etc/terminfo/g/gnome
etc/terminfo/g/gnome-256color
etc/terminfo/k/kitty
etc/terminfo/k/konsole
etc/terminfo/k/konsole-256color
etc/terminfo/k/konsole-linux
etc/terminfo/l/linux
etc/terminfo/p/putty
etc/terminfo/p/putty-256color
etc/terminfo/r/rxvt
etc/terminfo/r/rxvt-256color
etc/terminfo/s/screen
etc/terminfo/s/screen-256color
etc/terminfo/s/st-0.6
etc/terminfo/s/st-0.7
etc/terminfo/s/st-0.8
etc/terminfo/s/st-16color
etc/terminfo/s/st-256color
etc/terminfo/s/st-direct
etc/terminfo/s/sun
etc/terminfo/t/terminator
etc/terminfo/t/terminology
etc/terminfo/t/terminology-0.6.1
etc/terminfo/t/terminology-1.0.0
etc/terminfo/t/terminology-1.8.1
etc/terminfo/t/tmux
etc/terminfo/t/tmux-256color
etc/terminfo/v/vt100
etc/terminfo/v/vt102
etc/terminfo/v/vt200
etc/terminfo/v/vt220
etc/terminfo/v/vt52
etc/terminfo/v/vte
etc/terminfo/v/vte-256color
etc/terminfo/x/xterm
etc/terminfo/x/xterm-256color
etc/terminfo/x/xterm-color
etc/terminfo/x/xterm-xfree86
```

Although the `terminfo` entries number in base package is quit a few, comparing with `ncurses-terminfo` package, the base package is just a baby. To narrow down the entries number further, some entries take priority over others, such as `xterm-256color`, `alacritty`, `kitty`, `tmux-256color`, `putty-256color`.

## State sync design

### Real terminal send data to client

All the user input is send from real terminal emulator to (Aprish) local client. From `pty` master to slave.

### Client send user input to server

On the client side, client read from stand input device, which is a `pty` slave.

- The client reads the user input. [user keystroke]
- The client saves the user input in new state. [state saved in `UserStream`]
- When it's time to send state to server, [`network.tick()`]
- The client calculates the difference between new state and sent states. [`diff_from()` of `UserStream`]
- The calculated difference contains `resize` and `keystroke`.
- The client sends the difference to server through the network.

### Server update remote terminal.

On the server side, server receives new state from the network. Then server applies it to server terminal.

- The server receives the difference from client. [`recv()`]
- The server use the difference to rebuild the remote state. [`apply_string()` of `UserStream`]
- After the remote state is received, [`serve()`]
- The server applies the actions to the local terminal emulator. [`serve()`->`act()`]
  - For `keystroke` action, the server applies the input to terminal emulator.
  - For `resize` action, the server adjusts the size of terminal emulator.
- If there is any response from terminal emulator, the response will be sent back to host application.

![aprilsh.svg](../img/aprilsh.svg)

### Server send terminal difference to client

On the server side, server receives (terminal) application output from the `pty` master, as well as terminal write back.

- When it's time to send state to client, [`network.tick()`]
- The server calculates the difference between new state and sent states. [`diff_from()` of `Complete`]
- The calculated difference is a mix of escape sequences and data, which contains `echo ack`, `resize`, `mix`.
- The server sends the difference to the client through network.

### Client update local terminal

On the client side, client receives new state from the network. Then the client applies it to local terminal.

- The client receives the difference from server [`recv()`].
- The client applies `resize`, `mix` to the local terminal and save `echo ack`. [`apply_string()` of `Complete`]

Please note that there are two local termianls. One is used to track received terminal. The other is used to find difference for real terminal emulator, such as `kitty`, `alacritty` or `wezterm`. Here the local terminal changed is the received terminals.

### Client update real terminal

On the client side, the client synchronize the client terminal to real terminal emulator. Here the local terminal is used to find the difference for real terminal.

- When it's time to display the state to real termninal emulator [`output_new_frame()`]
- The client fetches the latest received terminal state.
- The client applies the prediction to the new state.
- The client calculates the difference between new state, prediction and local state. [`new_frame()` of `Display`]
  - Here the local state is the last used state.
- The calculated difference is a mix of escape sequences and data.
- The mix is written to the `pty` slave, which will output to the real terminal emulator.

Thus, we have the opportunity to use terminfo to output to the real terminal. Between the client and server, we are free to optimize the state synchronization.

## Terminal design

- [Understanding The Linux TTY Subsystem](https://ishuah.com/2021/02/04/understanding-the-linux-tty-subsystem/)
- [Build A Simple Terminal Emulator In 100 Lines of Golang](https://ishuah.com/2021/03/10/build-a-terminal-emulator-in-100-lines-of-go/)

## Reference

- `mosh` source code analysis [client](https://github.com/ericwq/examples/blob/main/tty/client.md), [server](https://github.com/ericwq/examples/blob/main/tty/server.md)
- [Unicode 14.0 Character Code Charts](http://www.unicode.org/charts/)
- [XTerm Control Sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
- [wezterm Escape Sequences](https://wezfurlong.org/wezterm/escape-sequences.html)
- [Linux man pages](https://linux.die.net/man/)
- [C++ Reference](http://www.cplusplus.com/reference/)
- [Linux logging guide: Understanding syslog, rsyslog, syslog-ng, journald](https://ikshitij.com/linux-logging-guide)
- [Benchmarking in Golang: Improving function performance](https://blog.logrocket.com/benchmarking-golang-improve-function-performance/)
- [Golang Field Alignment](https://medium.com/@didi12468/golang-field-alignment-2e657e87668a)
- [Structure size optimization in Golang (alignment/padding). More effective memory layout (linters)](https://itnext.io/structure-size-optimization-in-golang-alignment-padding-more-effective-memory-layout-linters-fffdcba27c61)

Need some time to figure out how to support CSI u in aprilsh.

- [Comprehensive keyboard handling in terminals](https://sw.kovidgoyal.net/kitty/keyboard-protocol/#functional-key-definitions)
- [feat(tui): query terminal for CSI u support](https://github.com/neovim/neovim/pull/18181)
- [Fix Keyboard Input on Terminals - Please](https://www.leonerd.org.uk/hacks/fixterms/)
- [xterm modified-keys](https://invisible-island.net/xterm/modified-keys.html)
