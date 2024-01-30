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
makedepends="
	go
	protoc
	utmps-dev
	musl-locales
	"
install=""
subpackages="$pkgname-client $pkgname-server"
source="https://github.com/ericwq/aprilsh/archive/refs/tags/$pkgver.tar.gz"
startdir="/home/packager/"
builddir="$srcdir"/$pkgname-$pkgname-$pkgver

prepare() {
	default_prepare
	printf "srcdir=${srcdir}\nstartdir=${startdir}\npkgdir=${pkgdir}\nbuilddir=${builddir}\n"
}

build() {
	# go protocol buffers plugin
	printf "download protoc-gen-go\n"
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	# install depends go module
	go mod tidy
	ls -al
	# use the following command to generate protocol buffer code
	printf "download protoc-gen-go\n"
	cd "$builddir"/protobufs
	protoc --go_out=. -I . "$builddir"/protobufs/transportInstruction.proto
	protoc --go_out=. -I . "$builddir"/hostInput.proto
	protoc --go_out=. -I . "$builddir"/protobufs/userInput.proto

	BuildVersion=`git describe --tags`
   ModuleName=`head ../../go.mod | grep "^module" | awk '{print $2}'`
   BuildTime=$(date "+%F %T")
   GoVersion=`go version | grep "version" | awk '{print $3,$4}'`
   GitCommit=`git rev-parse HEAD`
   GitBranch=`git rev-parse --abbrev-ref HEAD`

   echo "build server start: "$(date "+%F %T.")
	cd "$builddir"/frontend/server
   go build -ldflags="-s -w
      -X '${ModuleName}/frontend.BuildVersion=${BuildVersion}'
      -X '${ModuleName}/frontend.GoVersion=${GoVersion}'
      -X '${ModuleName}/frontend.GitCommit=${GitCommit}'
      -X '${ModuleName}/frontend.GitBranch=${GitBranch}'
      -X '${ModuleName}/frontend.BuildTime=${BuildTime}'" -o "$builddir"/bin/apshd .
   echo "build server end  : "$(date "+%F %T.")
   echo "output server to  : ~/.local/bin/apshd"

   echo "build client start: "$(date "+%F %T.")
	cd "$builddir"/frontend/client
   go build -ldflags="-s -w
      -X '${ModuleName}/frontend.BuildVersion=${BuildVersion}'
      -X '${ModuleName}/frontend.GoVersion=${GoVersion}'
      -X '${ModuleName}/frontend.GitCommit=${GitCommit}'
      -X '${ModuleName}/frontend.GitBranch=${GitBranch}'
      -X '${ModuleName}/frontend.BuildTime=${BuildTime}'" -o "$builddir"/bin/apsh .
   echo "build client end  : "$(date "+%F %T.")
   echo "output client to  : ~/.local/bin/apsh"
}

package() {
	install -Dm755 "$builddir"/bin/apshd "$pkgdir"/usr/bin
	install -Dm755 "$builddir"/bin/apsh  "$pkgdir"/usr/bin
}

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

sha512sums="
51499e579b92a51b4096893b9a2ec3f7c6af7d0ef232725d14176348abcecdd12b5bc3ed2beec510b7e18ee26eb856cd3c7ed05c255a7478ee1c1b63cd4e4494  0.5.6.tar.gz
"
