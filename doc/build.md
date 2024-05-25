## prepare container for apk building
create the container according to [Creating an Alpine package](https://wiki.alpinelinux.org/wiki/Creating_an_Alpine_package).
The container install `alpine-sdk sudo atools` packages, create `packager` user, generate abuild keys, and cache the aports fork by ericwq057.

```sh
docker build -t abuild:0.1.0 -f abuild.dockerfile .
docker build --no-cache --progress plain -t abuild:0.1.0 -f abuild.dockerfile .
```
if you update abuild keys, remember back up the keys.

run as root
```sh
docker run -u root --rm -ti -h abuild --env TZ=Asia/Shanghai --name abuild --privileged \
        --mount source=proj-vol,target=/home/ide/proj \
        --mount type=bind,source=/Users/qiwang/dev,target=/home/ide/develop \
        abuild:0.1.0
```
run as packager
```sh
docker run -u packager --rm -ti -h abuild --env TZ=Asia/Shanghai --name abuild --privileged \
        --mount source=proj-vol,target=/home/ide/proj \
        --mount type=bind,source=/Users/qiwang/dev,target=/home/ide/develop \
        abuild:0.1.0
```
### build apk files

if run as root, use `apk update` unlock the permission problem for abuild.
```sh
apk update
sudo -u packager sh
```
run as `packager` user, git pull aports fork.
```sh
sudo apk update
sudo apk add go protoc utmps-dev ncurses-terminfo openssh-client
sudo apk add musl-locales protoc-gen-go colordiff
cd ~/aports
# rebase pull
git config pull.rebase true     # rebase pull
git pull                        # get latest update
# delete old aprilsh branch
git branch -a                   # list all branches
git checkout aprilsh            # switch to branch
git checkout master             # switch to master
git branch -d aprilsh           # delete local branch
git push origin -d aprilsh      # delete remote branch
# create new branch and switch to it.
git branch aprilsh              # create branch
git checkout aprilsh            # switch to branch
```
<!-- https://www.freecodecamp.org/news/git-delete-remote-branch/ -->
create aprilsh directory if we don't have it.
```sh
ls ~/aports/testing/aprilsh
mkdir -p ~/aports/testing/aprilsh
cd ~/aports/testing/aprilsh
```
copy APKBUILD and other files from mount point. clean unused file.
```sh
cp /home/ide/develop/aprilsh/build/APKBUILD .
cp /home/ide/develop/aprilsh/build/apshd.* .
```
lint, checksum, build the apk.
```sh
apkbuild-lint APKBUILD
abuild checksum
abuild -r
```
### copy keys and apk to mount point
delete the old packages directory, note the `cp -r` command, it's important to keep the [directory structure of local repository](#directory-structure-of-local-repository).
```sh
rm -rf /home/ide/proj/packages                        # clean local repo/mount point
cd && cp -r packages/ /home/ide/proj/                 # copy apk to local repo/mount point
cp .abuild/packager-*.rsa.pub /home/ide/proj/packages # copy public key to mount point
```
### update apk to github pages
```sh
cd ~/packages/testing/x86_64
rm /home/ide/develop/ericwq.github.io/alpine/v3.19/testing/x86_64/*
cp * /home/ide/develop/ericwq.github.io/alpine/v3.19/testing/x86_64/
```
### validate tarball and apk
validate the apk content
```sh
cd ~/packages/testing/x86_64
tar tvvf aprilsh-0.5.49-r0.apk
```
validate the tarball.
```sh
cd /var/cache/distfiles
tar tvvf aprilsh-0.5.48.tar.gz
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
cd ~/aport/testing
git add aprilsh
git commit -a
git diff
git push origin aprilsh
```
## directory structure of alpine repository
if you don't keep the directory structure of alpine repository, you will get the following error:
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
### add local repository
install public key to local key store
```sh
cp /home/ide/proj/packages/packager-*.rsa.pub /etc/apk/keys
```
add local repository to apk repositories
```sh
echo "/home/ide/proj/packages/testing/" >> /etc/apk/repositories
```
### add remote repository
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
## install and validate apk files
update repositories metatdata, install new apk and restart apshd service.
```sh
apk update
rc-service apshd stop
apk del aprilsh
apk add aprilsh
rc-service apshd start
```
search aprilsh and validate the version.
```sh
apk search aprilsh
apsh -v
apshd -v
```
## add apshd service
```sh
rc-update add apshd boot
rc-service apshd start
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
apk add alpine-sdk sudo mandoc abuild-doc
adduser -D packager
addgroup packager abuild
echo 'packager ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/packager
sudo -u packager sh
abuild-keygen -n --append --install
```

generating a new apkbuild file with newapkbuild
```sh
newapkbuild \
    -n aprilsh \
    -d "Remote shell support intermittent or mobile network" \
    -l "MIT" \
    -a \
    "https://github.com/ericwq/aprilsh/archive/refs/tags/0.5.6.tar.gz"
```
