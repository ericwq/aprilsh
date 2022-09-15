# mosh network connection

## `Connection()`

- `Connection()` calls `key()` to generate a random key for the connection. `key()` is of type `Base64Key`.
- `Connection()` calls `session()` with the above key as parameter. `session()` is in charge of encrypt/decrypt.
- `Connection()` calls `setup()` to update the `last_port_choice` field with current time.
- `Connection()` calls `try_bind()` to bing to desired IP first, if we have `desired_ip` parameter; `try_bind()` return on success.
- `Connection()` calls `try_bind()` to try any local interface, return on sucess. see [next](#try_bind) section for detail.

### `try_bind()`

- `try_bind()` builds an instance for `AddrInfo` class with the following `addrinfo` as hint:
  - `getaddrinfo()` is called in the constructor of `AddrInfo` class to create struct `addrinfo`.
  - `hints.ai_family = AF_UNSPEC;`
  - `hints.ai_socktype = SOCK_DGRAM;`
  - `hints.ai_flags = AI_PASSIVE | AI_NUMERICHOST | AI_NUMERICSERV;`
  - `The Linux programming interface` Page 1213~1216
- `try_bind()` builds an instance for `Socket` class with the local address family as parameter.
  - `Socket()` calls `socket()` to create socket with the following options.
    - `SOCK_DGRAM` is the main parameter for the `socket()` system call.
    - `Advanced programming in the UNIX Environment` Page 600
  - `Socket()` calls `setsockopt()` to set socket options several times for different environment, such as:
    - `IPPROTO_IP, IP_MTU_DISCOVER`
    - `IPPROTO_IP, IP_TOS`
    - `IPPROTO_IP, IP_RECVTOS`
    - `UNIX Network Programming: The Sockets Networking API` Page 193,214
- `try_bind()` iterates from the low port to the high port number, for each port:
  - `try_bind()` calls `bind()` the socket to the address.
  - `UNIX Network Programming: The Sockets Networking API` Page 101,102
  - `try_bind()` calls `set_MTU()` to set MTU.
  - upon the socket is bind to the address and port, return.
  - otherwise, increase the port number, go through the next iteration.
- if the high port number is reached, `try_bind()` prints the error message and quits with error message.

# mosh client roaming

## receive a packet from client

- `recv()` receives a packet from remote.
- `recv()` iterates through each connection saved in `Connection`.
- `recv()` calls `recv_one()` for each connection to get the payload.
  - `recv_one()` a.k.a. `Connection:recv_one()`.
  - `recv_one()` receives a packet from client, the server checks the cached `remote_addr` field against the `packet_remote_addr` value.
  - `recv_one()` updates the `remote_addr` and `remote_addr_len` fields if it's different from the previous packet.
- `recv()` calls `prune_sockets()` to clean old socket.

## send packet to remote

- `send()` sends a packet to remote.
- `send()` creates a `Packet` based on the payload.
- `send()` encrypts the `Packet` message.
- `send()` calls `sendto()` to send the packet to the remote with the last socket in socket list.
- `send()` check the sent data size to check the error.
- for server side:
  - `send()` checks the `last_heard` time, if no contact since `SERVER_ASSOCIATION_TIMEOUT`, set `has_remote_addr` false.
- for client side:
  - `send()` checks the `last_port_choice` and `last_roundtrip_success`, if `PORT_HOP_INTERVAL` passed, calls `hop_port()`

## hop port

- `hop_port()` a.k.a. `Connection.hop_port()`.
- `hop_port()` calls `setup()` to update `last_port_choice`.
- `hop_port()` creates a new socket to the `remote_addr` and add it to the socket list in `Connection`.
- `hop_port()` calls `prune_sockets()` to clean old socket.

# go net package

- [Go: Deep dive into net package learning from TCP server](https://dev.to/hgsgtk/how-go-handles-network-and-system-calls-when-tcp-server-1nbd)
- [Socket sharding in Linux example with Go](https://dev.to/douglasmakey/socket-sharding-in-linux-example-with-go-4mi7)
- [Go socket design from user point](https://tonybai.com/2015/11/17/tcp-programming-in-golang/)
- [GopherCon 2019 - Socket to me: Where do Sockets live in Go?](https://about.sourcegraph.com/blog/go/gophercon-2019-socket-to-me-where-do-sockets-live-in-go)
- [golang wiki](https://github.com/golang/go/wiki/Articles)
- [深入 Go UDP 编程](https://colobu.com/2016/10/19/Go-UDP-Programming/#Read%E5%92%8CWrite%E6%96%B9%E6%B3%95%E9%9B%86%E7%9A%84%E6%AF%94%E8%BE%83)
- [详解 Go 语言 I/O 多路复用 netpoller 模型](https://www.luozhiyun.com/archives/439)

## `ListenConfig.ListenPacket()`

I choose `ListenConfig.ListenPacket()` because `ListenConfig` allow us to change socket configuration and parameters. `ListenConfig.ListenPacket()` calls `sl.listenUDP()` to get the listening socket.

- `sl.listenUDP()` a.k.a. `sysListener.listenUDP()`.
- `sysListener.listenUDP()` calls `internetSocket()` to get the socket file descriptor.
- `sysListener.listenUDP()` calls `internetSocket()` with `SOCK_DGRAM` as socket type parameter.
  - `internetSocket()` calls `favoriteAddrFamily()` to get the address family and `ipv6only` variable.
  - `internetSocket()` calls `socket()` to create the socket file descriptor. see [next](#socket) section for detail.
- `sysListener.listenUDP()` calls `newUDPConn()` to build a `UDPConn` type value.

### `socket()`

- `socket()` calls `sysSocket` to create the socket.
  - `sysSocket()` calls system call `socket()` and set `SOCK_NONBLOCK` and `SOCK_CLOEXEC` option for the socket.
- `socket()` calls `setDefaultSockopts` to set socket options.
  - `setDefaultSockopts()` calls system call `setsockopt()` and set `SOL_SOCKET` and `SO_BROADCAST`.
- `socket()` calls `fd.listenDatagram()` for `SOCK_DGRAM` when local addr is not nil but remote addr is nil.
- `socket()` calls `fd.listenStream()` for `SOCK_STREAM` when local addr is not nil but remote addr is nil. see [next](#listenstream) section for detail.
- `socket()` calls `fd.dial()` to initialize dialer socket file descriptor. see [next](#dial) section for detail.

### `listenStream()`

- `fd.listenStream()` a.k.a. `netFD.listenDatagram()`.
- `listenStream()` calls `laddr.sockaddr()` to convert local address to `syscall.Sockaddr` type.
- `listenStream()` calls `ctrlFn` defined in `ListenConfig` to setup the socket options.
- `listenStream()` calls system call `bind()` to bind the socket to the local address.
- `listenStream()` calls `fd.init()` to initialize the socket file descriptor for `netpoll`
- `listenStream()` calls `syscall.Getsockname()` to get the socket name.
- `listenStream()` calls `fd.addrFunc()` to convert the local address from `syscall.Sockaddr` type to `UDPAddr` type.
- `listenStream()` returns the socket file descriptor.

### `dial()`

- `fd.dial()` a.k.a. `netFD.dial()`.
- `dial()` calls `ctrlFn` defined in `ListenConfig` to setup the socket options.
- if there is local address:
  - `dial()` calls `laddr.sockaddr()` to convert local address to `syscall.Sockaddr` type.
  - `dial()` calls system call `bind()` to bind the socket to the local address.
- if there is remote address:
  - `dial()` calls `raddr.sockaddr()` to convert remote address to `syscall.Sockaddr` type.
  - `dial()` calls system call `connect()` to connect to the remote address and initialize the socket file descriptor for `netpoll`.
- `dial()` calls `syscall.Getsockname()` to get the socket name.
- `dial()` calls `fd.setAddr()` method to set the local and remote address field in `netFD`.
