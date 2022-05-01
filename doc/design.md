# Basic design

## Application output

On the server side, server receives application output from the `pty` device.

- The application output is a mix of escape sequence and data.
- The server reads the mix of escape sequence and data.
- The server apply the mix to terminal emulator on the server side.
- Here apply means to operate the terminal emulator according to the escape sequences.
- The server calculate the difference between new state and sent states.
- The calculated result is a mix of escape sequences and data.
- The server send the `echo ack`, `resize`, `mix` to the client.

On the client side, client receives new state from the network. The new state is the `framebuffer` with draw state inside `framebuffer`.

- The client apply (act) `resize`, `mix` to the local terminal emulator and save `echo ack`.
- The client apply the prediction to the new state.
- The client calculate the difference between the new state and local state.
- The calculated result is a mix of escape sequences and data.
- The mix is written to the client standard out, which will output to the local terminal emulator.
