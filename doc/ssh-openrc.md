## prepare ssh container for alpine
To avoid port conflict, the container map ssh port 22 to 8022 and map udp port 810* to 820*.

### build container
Run the following command to build ssh image, which perform the following actions:
- install openssh server, rsyslog.
- import local ssh rsa key into container.
- create user: eric.
- set password for root and eric user.
- transfer public key to `$HOME/.ssh/authorized_keys` for root and eric user.

```sh
cd aprilsh/build
docker build --build-arg ROOT_PWD=password \
        --build-arg USER_PWD=password \
        --build-arg SSH_PUB_KEY="$(cat ~/.ssh/id_rsa.pub)" \
        --progress plain -t openrc:0.1.0 -f openrc.dockerfile .
```

### start container
Run the following command to start ssh container, which perform the following action:
- mapping ssh port 22 to 8022,
- mapping udp port 810[0..3] to 820[0..3],
- set hostname and container name to `openrc`.

```sh
docker run --env TZ=Asia/Shanghai --tty --privileged \
    --volume /sys/fs/cgroup:/sys/fs/cgroup:rw \
    --hostname openrc --name openrc -d -p 8022:22 \
    -p 8201:8101/udp -p 8202:8102/udp -p 8203:8103/udp openrc:0.1.0
```
### check local ssh key
```sh
qiwang@Qi15Pro ~ % ls -al ~/.ssh
total 64
drwx------  10 qiwang  staff   320 May 16 09:25 .
drwxr-xr-x+ 36 qiwang  staff  1152 May 16 12:59 ..
-rw-------@  1 qiwang  staff   464 Feb 18 09:23 id_ed25519
-rw-r--r--@  1 qiwang  staff   102 Feb 18 09:23 id_ed25519.pub
-rw-------   1 qiwang  staff  2610 Feb  9  2022 id_rsa
-rw-r--r--   1 qiwang  staff   574 Feb  9  2022 id_rsa.pub
```
if you don't have any ssh keys, run the following command to generate it.
```sh
ssh-keygen -t ed25519
ssh-keygen -t rsa
```
### add rsa key to ssh agent
Here is my ssh version:
- ssh client: OpenSSH_9.0p1, LibreSSL 3.3.6
- ssh server: OpenSSH_9.6p1, OpenSSL 3.1.4 24 Oct 2023

if apsh reports `Failed to authenticate user "packager"`, which means your rsa key doen's work and sshd log shows: `Connection closed by authenticating user eric 192.168.65.1 port 22915 [preauth]`, which might means rsa key is too long, use ssh agent as work-around.
```sh
ssh-add ~/.ssh/id_rsa   # add rsa private key to agent
ssh-add -L              # check public key represented by the agent
```
### copy ssh public key to target host
```sh
ssh-copy-id -p 8022 -i ~/.ssh/id_rsa.pub eric@localhost
ssh-copy-id -p 8022 -i ~/.ssh/id_rsa.pub root@localhost
ssh-copy-id -p 8022 -i ~/.ssh/id_ed25519.pub eric@localhost
ssh-copy-id -p 8022 -i ~/.ssh/id_ed25519.pub root@localhost
```

### verified ssh authentication with public key.
if ssh reports: "WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!".
```sh
rm ~/.ssh/known_hosts*
```
now, ssh login to verify ssh works for you.
```sh
ssh -p 8022 root@localhost
ssh -p 8022 eric@localhost
```
### setup utmps service
```sh
setup-utmp
```
