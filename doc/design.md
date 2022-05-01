# Basic design

what we need is a terminal emulator, which can:

- accept and parse the mix of data and escape sequences and perform the action accordingly.
- caculate the difference between states of terminal emulator and ouput mix.
- apply the output mix to replicate the state of terminal emulator.

How to support multiple terminfo entry in terminal emulator, such as `xterm-256color`,
`rxvt-256color`, `alacritty` still need a solution.

## User input

On the client side, client read from stand input device, which is a `pty` device.

- The client reads the user input.
- The client saves the user input in new state.
- The client calculates the difference [`diff_from()`] betwee new state and sent states.
- The calculated result contains `resize` and `keystroke`.
- The client sends the `resize` and `keystroke` to server.

On the server side, server receives new state from the network.

- The server applies `resize` and `keystroke` to the local terminal emulator.
- For `keystroke`, the server applies [`act()`] the input to terminal emulator.
- FOr `resize`, the server adjusts the size of terminal emulator.
- If there is any response from terminal emulator, the response will be sent back to host.

## Application output

On the server side, server receives application output from the `pty` device.

- The application output is a mix of escape sequence and data.
- The server reads the mix of escape sequence and data.
- The server applies [`act()`] the mix to terminal emulator on the server side.
- If there is any response from terminal emulator, the response will be sent back to host.
- Here apply means to operate the terminal emulator according to the escape sequences and data.
- The server calculates the difference [`new_frame()`] between new state and sent states.
- The calculated result is a mix of escape sequences and data.
- The server send the `echo ack`, `resize`, `mix` to the client.

On the client side, client receives new state from the network.

- The client apply [`act()`] `resize`, `mix` to the local terminal emulator and save `echo ack`.
- The client apply the prediction to the new state.
- The client calculate the difference [`new_frame()`] between the new state and local state.
- The calculated result is a mix of escape sequences and data.
- The mix is written to the client standard out, which will output to the local terminal emulator.
