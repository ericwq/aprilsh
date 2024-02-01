## build and run container

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

## create the package
`apk update` unlock the permission problem for abuild.

```shell
$ apk update
# sudo -u packager sh
% cd
% mkdir -p aports/main/aprilsh
% cd ~/aports/main/aprilsh/
% cp /home/ide/develop/aprilsh/build/APKBUILD .
% abuild checksum
% abuild -r
% REPODEST=~/packages/3.19 abuild -r
% tar tvvf packages/main/x86_64/aprilsh-0.5.13-r0.apk
```

## install the package

```shell
% exit
# apk add /home/packager/packages/main/x86_64/aprilsh-server-0.5.13-r0.apk
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

## prepare docker environment
```sh
% docker run --rm -ti --privileged -h abuild --env TZ=Asia/Shanghai --name abuild \
        --mount source=proj-vol,target=/home/ide/proj \
        --mount type=bind,source=/Users/qiwang/dev,target=/home/ide/develop \
        alpine:3.19 
```

## setup your system and account
```sh 
# apk add alpine-sdk sudo mandoc abuild-doc
# adduser -D packager
# addgroup packager abuild
# echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager
# sudo -u packager sh
% abuild-keygen -n --append --install
```

## generating a new apkbuild file with newapkbuild
```sh
% newapkbuild \
    -n aprilsh \
    -d "Remote shell support intermittent or mobile network" \
    -l "MIT" \
    -a \
    "https://github.com/ericwq/aprilsh/archive/refs/tags/0.5.6.tar.gz"
```
