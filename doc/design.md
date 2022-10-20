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

- The client reads the user input [user keystroke].
- The client saves the user input in new state [state saved in `UserStream`]
- When it's time to send state to server [`network.tick()`]
- The client calculates the difference between new state and sent states. [`diff_from()` of `UserStream`]
- The calculated difference contains `resize` and `keystroke`.
- The client sends the difference to server through the network.

### Server update remote terminal.

On the server side, server receives new state from the network. Then server applies it to server terminal.

- The server receives the difference from client [`recv()`].
- The server use the difference to rebuild the remote state [`UserStream`].
- After received the remote state [`serve()`],
- The server applies the user kestroke to the local terminal emulator. [`apply_string()` of `UserStream`]
  - For `keystroke`, the server applies the input to terminal emulator. [`serve()`->`act()`]
  - For `resize`, the server adjusts the size of terminal emulator.
- If there is any response from terminal emulator, the response will be sent back to host application.

![aprilsh.svg](../img/aprilsh.svg)

### Server send terminal difference to client

On the server side, server receives (terminal) application output from the `pty` master, as well as terminal write back.

- When it's time to send state to client [`network.tick()`]
- The server calculates the difference between new state and sent states. [`diff_from()` of `Complete`]
- The calculated difference is a mix of escape sequences and data, which contains `echo ack`, `resize`, `mix`.
- The server sends the difference to the client through network.

### Client update local terminal

On the client side, client receives new state from the network. Then the client applies it to local terminal.

- The client receives the difference from server [`recv()`].
- The client applies `resize`, `mix` to the local terminal and save `echo ack`. [`apply_string()` of `Complete`]
- The client applies the prediction to the new state.

Please note that there are two local termianls. One is used to receive terminal state. One is used for display to
real terminal. Here the local terminal changed is the received terminals.

### Client update real terminal

On the client side, the client synchronize the client terminal to real terminal emulator. Here the terminal is used for computing difference for real terminal.

- When it's time to display the state to real termninal emulator [`output_new_frame()`]
- The client calculates the difference between new state, prediction and local state. [`new_frame()` of `Display`]
- The calculated difference is a mix of escape sequences and data.
- The mix is written to the `pty` slave, which will output to the real terminal emulator.

Thus, we have the opportunity to use terminfo to output to the real terminal. Between the client and server, we are free to optimize the state synchronization.
