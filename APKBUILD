# Contributor: Wang Qi <ericwq057@qq.com>
# Maintainer: Wang Qi <ericwq057@qq.com>
pkgname=aprilsh
pkgver=0.5.6
pkgrel=0
pkgdesc="Remote shell support intermittent or mobile network"
url="https://github.com/ericwq/aprilsh"
arch="all"
license="MIT"
depends="musl-locales utmps"
depends_dev="go protoc"
makedepends="$depends_dev utmps-dev musl-locales"
checkdepends=""
install=""
subpackages="$pkgname-dev $pkgname-doc"
source="https://github.com/ericwq/aprilsh/archive/refs/tags/$pkgver.tar.gz"
builddir="$srcdir/$pkgname-$pkgver"

# _giturl="https://github.com/ericwq/aprilsh"
# _gittag="$pkgver"
# disturl="https://github.com/ericwq/aprilsh/archive/refs/tags/"
#
# snapshot() {
# 	mkdir -p "$srcdir"
# #	printf "path: ${SRCDEST:-$srcdir}\n"
# 	cd "${SRCDEST:-$srcdir}"
# 	if ! [ -d $pkgname.git ]; then
# 		git clone --bare  $_giturl || return 1
# 		cd $pkgname.git
# 	else
# 		cd $pkgname.git
# 		git fetch || return 1
# 	fi
#
# 	git archive --prefix=$pkgname/ -o "$SRCDEST"/$pkgname-$pkgver.tar.gz $_gittag
# #	scp "$SRCDEST"/$pkgname-$pkgver.tar.gz dev.alpinelinux.org:/archive/$pkgname/
# }

prepare() {
	default_prepare
	# go protocol buffers plugin
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	# install depends go module
	go mod tidy
	# use the following command to generate protocol buffer code
	cd ./protobufs
	protoc --go_out=. -I . ./protobufs/transportInstruction.proto
	protoc --go_out=. -I . ./hostInput.proto
	protoc --go_out=. -I . ./protobufs/userInput.proto
}

build() {
	# ./configure \
	# 	--build=$CBUILD \
	# 	--host=$CHOST \
	# 	--prefix=/usr \
	# 	--sysconfdir=/etc \
	# 	--mandir=/usr/share/man \
	# 	--localstatedir=/var
	# make
}

check() {
	# make check
}

package() {
	# make DESTDIR="$pkgdir" install
}

sha512sums="
57a73658ecb947af9dfad7a5e2931660ad1b8fa61d36c803c373e8aba13e9afa8398c1522765f5ea2b5df87d942cea17062faf30f589afa6acc744ff3ae4a409  utmps-0.1.2.2.tar.gz
"
