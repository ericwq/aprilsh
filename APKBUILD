# Contributor: Wang Qi <ericwq057@qq.com>
# Maintainer: Wang Qi <ericwq057@qq.com>
pkgname=aprilsh
pkgver=0.5.14
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
	"
install=""
subpackages=""
subpackages="$pkgname-client $pkgname-server"
source="$pkgname-$pkgver.tar.gz::https://github.com/ericwq/aprilsh/archive/refs/tags/$pkgver.tar.gz"
builddir="$srcdir"/$pkgname-$pkgver

export PATH=$PATH:~/go/bin
export GOCACHE="${GOCACHE:-"$srcdir/go-cache"}"
export GOTMPDIR="${GOTMPDIR:-"$srcdir"}"
export GOMODCACHE="${GOMODCACHE:-"$srcdir/go"}"

prepare() {
   # startdir="/home/packager/aports/main/aprilsh"
   # pkgdir="/home/packager/packages/"
	# mkdir -p "./packages"
	# mkdir -p "./aprilsh"
	printf "startdir=${startdir}\n"
	printf "srcdir  =${srcdir}\n"
	printf "builddir=${builddir}\n"
	printf "pkgdir  =${pkgdir}\n"
	# printf "PATH=$PATH\n"
	default_prepare
}

build() {
	# cd ${srcdir}
	# install go protocol buffers plugin
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	# install depends go module
	go mod tidy
	# use the following command to generate protocol buffer code
	protoc --go_out=. -I . ./protobufs/transportInstruction.proto
	protoc --go_out=. -I . ./protobufs/hostInput.proto
	protoc --go_out=. -I . ./protobufs/userInput.proto

	# prepare build info
	_BuildVersion=`head build.info | grep "tag:" | awk '{print $2}'`
   _ModuleName=`head ./go.mod | grep "^module" | awk '{print $2}'`
   _BuildTime=`date "+%F %T"`
   _GoVersion=`go version | grep "version" | awk '{print $3,$4}'`
   _GitCommit=`head build.info | grep "commit:" | awk '{print $2}'`
   _GitBranch=`head build.info | grep "branch:" | awk '{print $2}'`

   echo "build server start: `date '+%F %T'`"
   go build -ldflags="-s -w
      -X '${_ModuleName}/frontend.BuildVersion=${_BuildVersion}'
      -X '${_ModuleName}/frontend.GoVersion=${_GoVersion}'
      -X '${_ModuleName}/frontend.GitCommit=${_GitCommit}'
      -X '${_ModuleName}/frontend.GitBranch=${_GitBranch}'
      -X '${_ModuleName}/frontend.BuildTime=${_BuildTime}'" -o "${builddir}/bin/apshd" ./frontend/server/*.go
   echo "build server end  : `date '+%F %T'`"
   echo "output server to  : ${builddir}/bin/apshd"

   echo "build client start: `date '+%F %T'`"
   go build -ldflags="-s -w
      -X '${_ModuleName}/frontend.BuildVersion=${_BuildVersion}'
      -X '${_ModuleName}/frontend.GoVersion=${_GoVersion}'
      -X '${_ModuleName}/frontend.GitCommit=${_GitCommit}'
      -X '${_ModuleName}/frontend.GitBranch=${_GitBranch}'
      -X '${_ModuleName}/frontend.BuildTime=${_BuildTime}'" -o "${builddir}/bin/apsh" ./frontend/client/*.go
   echo "build client end  : `date '+%F %T'`"
   echo "output client to  : ${builddir}/bin/apsh"
}

check() {
	# cd ${srcdir}
	go test ./encrypt/...
	go test ./frontend/...
	go test ./network/...
	go test ./statesync/...
	# go test ./terminal/...
	# go test ./util/...
}

package() {
	install -Dm755 "$builddir/bin/apshd" "$pkgdir/usr/bin/apshd"
	install -Dm755 "$builddir/bin/apsh"  "$pkgdir/usr/bin/apsh"
}

client() {
	replaces="$pkgname"
	pkgdesc="$pkgname server"
	depends="musl-locales"
	mkdir -p "$subpkgdir/usr/bin"
	cp "$pkgdir/usr/bin/apsh" "$subpkgdir/usr/bin/"
}

server() {
	replaces="$pkgname"
	pkgdesc="$pkgname server"
	depends="musl-locales utmps"
	mkdir -p "$subpkgdir/usr/bin"
	cp "$pkgdir/usr/bin/apshd" "$subpkgdir/usr/bin/"
}

sha512sums="
d934ee2dcdb46dbd8793ba89661c7de0369af4c6845d0d5acb7c3f609a4be9d87561a79e681b41c6ee4bdf31715620071100c400a13cc1f8bd06a8c0563450a3  aprilsh-0.5.14.tar.gz
"
