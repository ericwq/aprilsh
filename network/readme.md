# mosh C++ code

## `try_bind()`

- `try_bind()` calls `getaddrinfo()` function to create struct `addrinfo`.
  - ` hints.ai_family = AF_UNSPEC;`
  - ` hints.ai_socktype = SOCK_DGRAM;`
  - ` hints.ai_flags = AI_PASSIVE | AI_NUMERICHOST | AI_NUMERICSERV;`
  - `The Linux programming interface` Page 1213~1216
- `try_bind()` calls `Socket()` to create socket for connection.
  - `Socket()` calls system call `socket` to create socket with the following options.
    - `Advanced programming in the UNIX Environment` Page 600
  - `Socket()` calls `setsockopt()` to set socket options
    - `IPPROTO_IPV6, IPV6_V6ONLY`
  - `IPPROTO_IP, IP_TOS`
  - `IPPROTO_IP, IP_RECVTOS`
  - `UNIX Network Programming: The Sockets Networking API` Page 193,214
- `try_bind()` iterates fromt the low port to the high port number. For each port:
  - `try_bind()` calls system call `bind()` the socket to the address.
    - `UNIX Network Programming: The Sockets Networking API` Page 101,102
  - `try_bind()` calls `set_MTU()` to set MTU.
  - upon the socket is bind to the address and port. just return.

# go net package

[Go: Deep dive into net package learning from TCP server](https://dev.to/hgsgtk/how-go-handles-network-and-system-calls-when-tcp-server-1nbd)

## `tryBind()`
