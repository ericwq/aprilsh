## prepare container for ssh

build port container, which perform the following actions:

- set password for root and ide user.
- transfer public key to `$HOME/.ssh/authorized_keys` for root and ide user.

```sh
cd aprilsh/build
docker build --build-arg ROOT_PWD=password \
        --build-arg USER_PWD=password \
        --build-arg SSH_PUB_KEY="$(cat ~/.ssh/id_rsa.pub)" \
        --progress plain -t openrc:0.1.0 -f openrc.dockerfile .
```

start port container, which perform the following action:

- mapping tcp port 22 to 8022, mapping udp port 810[0..3] to 820[0..3].
- mount docker volume `proj-vol` to `/home/ide/proj`.
- mount local directory `/Users/qiwang/dev` to `/home/ide/develop/`.
- set hostname and container name to `openrc-port`.

```sh
docker run --env TZ=Asia/Shanghai --tty --privileged \
    --volume /sys/fs/cgroup:/sys/fs/cgroup:rw \
    --hostname openrc --name openrc -d -p 22:22 \
    -p 8101:8101/udp -p 8102:8102/udp -p 8103:8103/udp openrc:0.1.0
```

copy ssh public key to target host.
```sh
ssh-copy-id -i ~/.ssh/id_rsa.pub eric@localhost
ssh-copy-id -i ~/.ssh/id_ed25519.pub eric@localhost
```

verified you can login with the above public key.
```sh
ssh eric@localhost
```
