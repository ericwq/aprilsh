## prepare docker environment for apk building

```sh
% docker build -t abuild:0.1.0 -f abuild.dockerfile .
% docker build --no-cache --progress plain -t abuild:0.1.0 -f abuild.dockerfile .
% docker run -u root --rm -ti -h abuild --env TZ=Asia/Shanghai --name abuild --privileged \
        --mount source=proj-vol,target=/home/ide/proj \
        --mount type=bind,source=/Users/qiwang/dev,target=/home/ide/develop \
        abuild:0.1.0
% docker exec -ti --privileged abuild:0.1.0
% docker exec -u root -it abuild ash
```

## build alpine apk file

`apk update` unlock the permission problem for abuild.

```shell
$ apk update
```

switch to user packager and prepare the environment.
```shell
# sudo -u packager sh
% cd
% mkdir -p aports/main/aprilsh
% cd ~/aports/main/aprilsh/
```

get the APKBUILD and local file from mount point.
```shell
% cp /home/ide/develop/aprilsh/build/* .
% abuild checksum
```

build the apk.
```shell
% abuild -r
% REPODEST=~/packages/3.19 abuild -r
```

validate the tarball.
```shell
% cd /var/cache/distfiles
% tar tvvf aprilsh-0.5.48.tar.gz
```

copy keys and apk to mount point, validate the apk content
```shell
% cd .abuild/
% cp packager-65bd9c2a.rsa.pub /home/ide/proj/apk
% cd ~/packages/main/x86_64
% cp *.apk /home/ide/proj/apk
% tar tvvf packages/main/x86_64/aprilsh-0.5.13-r0.apk
```

## prepare docker environment for apk testing

start a new container.

```sh
% docker run --rm -ti --privileged -h abuild-test --env TZ=Asia/Shanghai --name abuild-test \
        --mount source=proj-vol,target=/home/ide/proj \
        --mount type=bind,source=/Users/qiwang/dev,target=/home/ide/develop \
        alpine:3.19
```

install the key from mount point.

```shell
# apk update
# apk add tzdata
# cp /home/ide/proj/apk/packager-65bd9c2a.rsa.pub /etc/apk/keys
```

## test alpine apk file
install the package and validate the program.

```shell
# cd /home/ide/proj/apk
# apk add aprilsh-0.5.48-r0.apk
# apsh -v
# apshd -v
```

## reference

- [How to build and install Alpine Linux package with newapkbuild](https://www.educative.io/answers/how-to-build-and-install-alpine-linux-package-with-newapkbuild)
- [Setting up a packaging environment for Alpine Linux (introducing alpkg)](https://blog.orhun.dev/alpine-packaging-setup/)
- [Creating an Alpine package](https://wiki.alpinelinux.org/wiki/Creating_an_Alpine_package)
- [APKBUILD examples](https://wiki.alpinelinux.org/wiki/APKBUILD_examples)
- [APKBUILD Reference](https://wiki.alpinelinux.org/wiki/APKBUILD_Reference#Examples)
- [mosh APKBUILD](https://gitlab.alpinelinux.org/alpine/aports/-/blob/master/main/mosh/APKBUILD)
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
