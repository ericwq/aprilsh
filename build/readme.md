## prepare container for apk building
create the container according to [Creating an Alpine package](https://wiki.alpinelinux.org/wiki/Creating_an_Alpine_package).
The container install `alpine-sdk sudo atools` packages, create `packager` user, generate abuild keys, and cache the aports fork by ericwq057.

```sh
% docker build -t abuild:0.1.0 -f abuild.dockerfile .
% docker build --no-cache --progress plain -t abuild:0.1.0 -f abuild.dockerfile .
```
run as root
```sh
% docker run -u root --rm -ti -h abuild --env TZ=Asia/Shanghai --name abuild --privileged \
        --mount source=proj-vol,target=/home/ide/proj \
        --mount type=bind,source=/Users/qiwang/dev,target=/home/ide/develop \
        abuild:0.1.0
```
run as packager
```sh
% docker run -u packager --rm -ti -h abuild --env TZ=Asia/Shanghai --name abuild --privileged \
        --mount source=proj-vol,target=/home/ide/proj \
        --mount type=bind,source=/Users/qiwang/dev,target=/home/ide/develop \
        abuild:0.1.0

```
### build apk files

if run as root, use `apk update` unlock the permission problem for abuild.

```sh
# apk update
# sudo -u packager sh
```

run as `packager` user, git pull aports fork.
```sh
% sudo apk update
% sudo apk add go protoc utmps-dev ncurses ncurses-terminfo musl-locales protoc-gen-go colordiff
% cd ~/aports
% git config pull.rebase true
% git pull
% git checkout aprilsh ## switch to branch
% git branch aprilsh   ## create branch
```

create aprilsh directory if we don't have it.
```sh
% ls ~/aports/testing/aprilsh
% mkdir -p ~/aports/testing/aprilsh
% cd ~/aports/testing/aprilsh
```

copy APKBUILD and other files from mount point. clean unused file.
```sh
% cp /home/ide/develop/aprilsh/build/* .
% rm *.dockerfile readme.md
```

lint, checksum, build the apk.
```sh
% apkbuild-lint APKBUILD
% abuild checksum
% abuild -r
```

### copy keys and apk to local repo
delete the old packages directory, note the `cp -r` command, it's important to keep the [directory structure of local repository](#directory-structure-of-local-repository).
```sh
% rm -rf /home/ide/proj/packages
% cd && cp -r packages/ /home/ide/proj/
% cp .abuild/packager-*.rsa.pub /home/ide/proj/packages
```

### validate tarball and apk
validate the apk content
```sh
% cd ~/packages/testing/x86_64
% tar tvvf aprilsh-0.5.49-r0.apk
```

validate the tarball.
```sh
% cd /var/cache/distfiles
% tar tvvf aprilsh-0.5.48.tar.gz
```

### commit the update to branch

Use the following commit message template for new aports (without the comments):
```txt
testing/aprilsh: new aport

https://github.com/ericwq/aprilsh
Remote shell support intermittent or mobile network
```
commit the update, push to the remote repository.
```sh
% git commit -a
% git diff
% git push origin aprilsh
```
## prepare openrc-nvide container for apk testing

please check [Run the openrc-nvide container](https://github.com/ericwq/nvide?tab=readme-ov-file#run-the-openrc-nvide-container) for detail.
```sh
$ docker run --env TZ=Asia/Shanghai --tty --privileged --volume /sys/fs/cgroup:/sys/fs/cgroup:rw \
    --mount source=proj-vol,target=/home/ide/proj \
    --mount type=bind,source=/Users/qiwang/dev,target=/home/ide/develop \
    -h openrc-nvide --name openrc-nvide -d -p 22:22  -p 8100:8100/udp  -p 8101:8101/udp -p 8102:8102/udp \
    -p 8103:8103/udp openrc-nvide:0.10.2
```

don't forget to start utmps service, install timezone package if needed.
```sh
# setup-utmp
# apk add tzdata openrc
```

and install package key for local repo
```sh
cp /home/ide/proj/packages/packager-*.rsa.pub /etc/apk/keys
```

add local repo to apk repositories
```sh
echo "/home/ide/proj/packages/testing/" >> /etc/apk/repositories
```

update repositories metatdata, install new apk and restart apshd service.
```sh
# apk update
# rc-service apshd stop
# apk del aprilsh
# apk add aprilsh
# rc-service apshd start
```

### directory structure of local repository
if you don't keep the directory structure of local repository, you will get the following error:
```sh
~ # apk update
WARNING: opening /home/ide/proj/packages/: No such file or directory
```

The local repository should contains `x86_64` directory and `APKINDEX.tar.gz` file:
```sh
# tree /home/ide/proj/packages/main
/home/ide/proj/packages/main
└── x86_64
    ├── APKINDEX.tar.gz
    ├── aprilsh-0.5.49-r0.apk
    ├── aprilsh-client-0.5.49-r0.apk
    └── aprilsh-openrc-0.5.49-r0.apk

1 directories, 4 files
```

### validate apk files
install the package and validate the program.

```sh
# apk search aprilsh
# apk add aprilsh
# apsh -v
# apshd -v
```
### add apshd service

```sh
# rc-update add apshd boot
# rc-service apshd start
```
## prepare container for port testing

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
docker run --env TZ=Asia/Shanghai --tty --privileged --volume /sys/fs/cgroup:/sys/fs/cgroup:rw \
    --mount source=proj-vol,target=/home/ide/proj \
    --mount type=bind,source=/Users/qiwang/dev,target=/home/ide/develop \
    --hostname openrc-port --name openrc-port -d -p 8022:22 \
    -p 8201:8101/udp -p 8202:8102/udp -p 8203:8103/udp openrc:0.1.0
```
## add remote repository
After you verified the local repository, you can serve the repository with github pages.
```sh
git clone https://github.com/ericwq/ericwq.github.io.git
cd ~/develop/ericwq.github.io/
```

copy keys and apks to codeberg pages
```sh
cp ~/.abuild/packager-663ebf9b.rsa /home/ide/develop/ericwq.github.io/alpine/
cp -r testing/ /home/ide/develop/ericwq.github.io/alpine/v3.19/
```

add our repository to /etc/apk/repositories
```sh
echo "https://ericwq.github.io/alpine/v3.19/testing" >> /etc/apk/repositories
```
download and store our signing key to /etc/apk/keys
```sh
wget -P /etc/apk/keys/ https://ericwq.github.io/alpine/packager-663ebf9b.rsa.pub
```

## add ssh public key to remote server

```sh
ssh-keygen -t ed25519
ssh-copy-id -i ~/.ssh/id_ed25519.pub root@localhost
ssh-copy-id -i ~/.ssh/id_ed25519.pub ide@localhost
ssh-add ~/.ssh/id_ed25519
```

## reference

- [How to build and install Alpine Linux package with newapkbuild](https://www.educative.io/answers/how-to-build-and-install-alpine-linux-package-with-newapkbuild)
- [Setting up a packaging environment for Alpine Linux (introducing alpkg)](https://blog.orhun.dev/alpine-packaging-setup/)
- [Creating an Alpine package](https://wiki.alpinelinux.org/wiki/Creating_an_Alpine_package)
- [APKBUILD examples](https://wiki.alpinelinux.org/wiki/APKBUILD_examples)
- [APKBUILD Reference](https://wiki.alpinelinux.org/wiki/APKBUILD_Reference#Examples)
- [mosh APKBUILD](https://gitlab.alpinelinux.org/alpine/aports/-/blob/master/main/mosh/APKBUILD)
- [Building / consuming alpine Linux packages inside containers and images](https://itsufficient.me/blog/alpine-build)
- [Alpine Package Keeper](https://wiki.alpinelinux.org/wiki/Alpine_Package_Keeper)
- [Working with the Alpine Package Keeper (apk)](https://docs.alpinelinux.org/user-handbook/0.1a/Working/apk.html)
- [Alpine Linux in a chroot](https://wiki.alpinelinux.org/wiki/Alpine_Linux_in_a_chroot)
- [How to Build an Alpine Linux Package](https://www.matthewparris.org/build-an-alpine-package/)
- [How to create a Bash completion script](https://opensource.com/article/18/3/creating-bash-completion-script)
- [Alpine Linux: New APKBUILD Workflow](https://thiagowfx.github.io/2022/01/alpine-linux-new-apkbuild-workflow/)
- [Can GitHub actions directly edit files in a repository?](https://github.com/orgs/community/discussions/25234)
- [todo_updater](https://github.com/logankilpatrick/TODO-List-Updater/blob/master/.github/workflows/todo_updater.yml)
- [github-push-action](https://github.com/ad-m/github-push-action)
- [workflow permission](https://github.com/ericwq/aprilsh/settings/actions)

## git source
```
_giturl="https://github.com/ericwq/aprilsh"
_gittag="$pkgver"
disturl="https://github.com/ericwq/aprilsh/archive/refs/tags/"

snapshot() {
	mkdir -p "$srcdir"
	cd "${SRCDEST:-$srcdir}"
	if ! [ -d $pkgname.git ]; then
		git clone --bare  $_giturl || return 1
		cd $pkgname.git
	else
		cd $pkgname.git
		git fetch || return 1
	fi

	git archive --prefix=$pkgname/ -o "$SRCDEST"/$pkgname-$pkgver.tar.gz $_gittag
	scp "$SRCDEST"/$pkgname-$pkgver.tar.gz dev.alpinelinux.org:/archive/$pkgname/
}
```

## manually setup your system and account
```sh 
# apk add alpine-sdk sudo mandoc abuild-doc
# adduser -D packager
# addgroup packager abuild
# echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager
# sudo -u packager sh
% abuild-keygen -n --append --install
```

generating a new apkbuild file with newapkbuild
```sh
% newapkbuild \
    -n aprilsh \
    -d "Remote shell support intermittent or mobile network" \
    -l "MIT" \
    -a \
    "https://github.com/ericwq/aprilsh/archive/refs/tags/0.5.6.tar.gz"
```
