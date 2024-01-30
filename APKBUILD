# Contributor: Wang Qi <ericwq057@qq.com>
# Maintainer: Wang Qi <ericwq057@qq.com>
pkgname=aprilsh
pkgver=0.5.13
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
# subpackages="$pkgname-client $pkgname-server"
# source="$pkgname-$pkgver.tar.gz::https://github.com/ericwq/aprilsh/archive/refs/tags/$pkgver.tar.gz"
source="$pkgname-$pkgver.tar.gz::https://github.com/ericwq/aprilsh/releases/download/$pkgver/$pkgname-$pkgver-linux-x64-musl.tar.gz"
# srcdir="/home/packager/aprilsh"
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
	printf "PATH=$PATH\n"
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
      -X '${_ModuleName}/frontend.BuildTime=${_BuildTime}'" -o "$builddir"/bin/apshd ./frontend/server/*.go
   echo "build server end  : `date '+%F %T'`"
   echo "output server to  : ${builddir}/bin/apshd"

   echo "build client start: `date '+%F %T'`"
	# cd "$builddir"/frontend/client
   go build -ldflags="-s -w
      -X '${_ModuleName}/frontend.BuildVersion=${_BuildVersion}'
      -X '${_ModuleName}/frontend.GoVersion=${_GoVersion}'
      -X '${_ModuleName}/frontend.GitCommit=${_GitCommit}'
      -X '${_ModuleName}/frontend.GitBranch=${_GitBranch}'
      -X '${_ModuleName}/frontend.BuildTime=${_BuildTime}'" -o "$builddir"/bin/apsh ./frontend/client/*.go
   echo "build client end  : `date '+%F %T'`"
   echo "output client to  : ${builddir}/bin/apsh"
}

check() {
	cd ${srcdir}
	go test ./encrypt/...
	go test ./frontend/...
	go test ./network/...
	go test ./statesync/...
	# go test ./terminal/...
	# go test ./util/...
}

package() {
	# mkdir -p "$pkgdir"/usr/bin/
	install -Dm755 "$builddir"/bin/apshd "$pkgdir"/usr/bin/apshd
	install -Dm755 "$builddir"/bin/apsh  "$pkgdir"/usr/bin/apsh
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
c82e4a6893c21ecf798629cdb525c55b70eec8c56e2ec1b4f23800ecba1832cf2b916901e5e0f12d9d195b34d424fbd8867b9c24ae2cb889095233da6fb22acc  aprilsh-0.5.13.tar.gz
"
