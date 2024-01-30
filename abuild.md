## prepare docker environment


## setup your system and account
```sh 
#apk add alpine-sdk sudo mandoc abuild-doc
#adduser -D packager
#addgroup packager abuild
#echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager
#sudo -u packager sh
%abuild-keygen -n --append --install
```

## generating a new apkbuild file
```sh
% newapkbuild \
    -n aprilsh \
    -d "Remote shell support intermittent or mobile network" \
    -l "MIT" \
    -a \
    "https://skarnet.org/software/utmps/utmps-0.1.2.2.tar.gz"
    && "https://github.com/ericwq/aprilsh/archive/refs/tags/aprilsh-0.5.2.tar.xz"
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