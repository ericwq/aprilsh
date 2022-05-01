# Basic design

What we need is a terminal emulator, which can:

- accept and parse the mix of data and escape sequences and perform the action accordingly.
- calculate the difference between states of terminal emulator and output mix.
- apply the output mix to replicate the state of terminal emulator.

In real world, the user may choose different terminal emulator. The `TERM` environment variable should be propagated from client to the server, just like SSH does. Different `TERM` means different terminal capability.

How to adapt to this situation: support different terminal emulator, such as `xterm-256color`,
`rxvt-256color`, `alacritty`, still need a solution.

- First solution: write separate independent parser and actor for different `terminfo` entry. The different implementation share the same terminal emulator data structure. The implementation is chosen by `TERM` at runtime.
- Second solution: write a base class which can parse and act on a base `terminfo` entry, which could be `st-256color`. (We don't choose `xterm-256color` because `xterm` is too complex) The other `terminfo` entry is implemented by a extend class. The different implementation share the same terminal emulator data structure. The implementation is chosen by `TERM` at runtime.

Both solution suggest we can support `terminfo` entries one by one, step by step. The scope of `terminfo` entries can be narrowed down to `ncurses-terminfo-base` package (the alpine linux platform).

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

## User input

On the client side, client read from stand input device, which is a `pty` slave.

- The client reads the user input.
- The client saves the user input in new state.
- The client calculates the difference between new state and sent states. [`diff_from()`]
- The calculated difference contains `resize` and `keystroke`.
- The client sends the difference to server through the network.

On the server side, server receives new state from the network.

- The server receives the difference from client.
- The server applies difference to the local terminal emulator.
- For `keystroke`, the server applies the input to terminal emulator. [`act()`]
- For `resize`, the server adjusts the size of terminal emulator.
- If there is any response from terminal emulator, the response will be sent back to host application.

![aprilsh.svg](../img/aprilsh.svg)

## Application output

On the server side, server receives application output from the `pty` master.

- The application output is a mix of escape sequence and data.
- The server reads the mix of escape sequence and data.
- The server applies the mix to terminal emulator on the server side. [`act()`]
- If there is any response from terminal emulator, the response will be sent back to host application.
- Here apply means to operate the terminal emulator according to the escape sequences and data.
- The server calculates the difference between new state and sent states. [`new_frame()`]
- The calculated difference is a mix of escape sequences and data, which contains `echo ack`, `resize`, `mix`.
- The server sends the difference to the client through network.

On the client side, client receives new state from the network.

- The client receives the difference from server.
- The client applies `resize`, `mix` to the local terminal emulator and save `echo ack`. [`act()`]
- The client applies the prediction to the new state.
- The client calculates the difference between the new state and local state. [`new_frame()`]
- The calculated difference is a mix of escape sequences and data.
- The second difference is mainly used to update the real terminal emulator display.
- The mix is written to the `pty` slave, which will output to the real terminal emulator.
