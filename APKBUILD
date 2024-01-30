# Contributor: Wang Qi <ericwq057@qq.com>
# Maintainer: Wang Qi <ericwq057@qq.com>
pkgname=aprilsh
pkgver=0.5.9
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
source="$pkgname-$pkgver.tar.gz::https://github.com/ericwq/aprilsh/archive/refs/tags/$pkgver.tar.gz"
# srcdir="~/aprilsh"
builddir="$srcdir"/$pkgname-$pkgver

prepare() {
	default_prepare
   # startdir="~/aports/main/aprilsh"
   # pkgdir="~/pkg/"
	printf "srcdir=${srcdir}\nstartdir=${startdir}\npkgdir=${pkgdir}\nbuilddir=${builddir}\n"
}

build() {
	# ls -al
	# go protocol buffers plugin
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	# install depends go module
	go mod tidy
	# use the following command to generate protocol buffer code
	# printf "***** start protoc build\n"
	# printf "$PATH\n"
	# printf "***** start protoc build\n"
	protoc --go_out=. -I . ./protobufs/transportInstruction.proto
	protoc --go_out=. -I . ./protobufs/hostInput.proto
	protoc --go_out=. -I . ./protobufs/userInput.proto
	ls -al

	BuildVersion=`head build.txt | grep "version:" | awk '{print $2}'`
   ModuleName=`head ./go.mod | grep "^module" | awk '{print $2}'`
   BuildTime=`date "+%F %T"`
   GoVersion=`go version | grep "version" | awk '{print $3,$4}'`
   GitCommit=`head build.txt | grep "commit:" | awk '{print $2}'`
   GitBranch=`head build.txt | grep "branch:" | awk '{print $2}'`

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
36ef95b5925dcc1f56da8c1d417110127a754eb93ccd7d62875de454798952eb1e9b410a272452ba72ffb1b5c5043c98ce986b1442c243f59af32f0eef8a53c0  aprilsh-0.5.8.tar.gz
"
