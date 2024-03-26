# Copyright 2022~2024 wangqi. All rights reserved.
# Use of this source code is governed by a MIT-style
# license that can be found in the LICENSE file.
# Packager: Wang Qi <ericwq057@qq.com>

# To Install:
#
# sudo yum -y install rpmdevtools && rpmdev-setuptree
# wget https://raw.github.com/nmilford/rpm-etcd/master/etcd.spec -O ~/rpmbuild/SPECS/etcd.spec
# wget https://github.com/coreos/etcd/releases/download/v2.0.9/etcd-v2.0.9-linux-amd64.tar.gz -O ~/rpmbuild/SOURCES/etcd-v2.0.9-linux-amd64.tar.gz
# wget https://raw.github.com/nmilford/rpm-etcd/master/etcd.initd -O ~/rpmbuild/SOURCES/etcd.initd
# wget https://raw.github.com/nmilford/rpm-etcd/master/etcd.sysconfig -O ~/rpmbuild/SOURCES/etcd.sysconfig
# wget https://raw.github.com/nmilford/rpm-etcd/master/etcd.nofiles.conf -O ~/rpmbuild/SOURCES/etcd.nofiles.conf
# wget https://raw.github.com/nmilford/rpm-etcd/master/etcd.logrotate -O ~/rpmbuild/SOURCES/etcd.logrotate
# rpmbuild -bb ~/rpmbuild/SPECS/etcd.spec

# centos7: protobuf-compiler only support proto2
Name:	  aprilsh
Version:  0.6.39
Release:  1%{?dist}
Summary:  Remote shell support intermittent or mobile network
License:  MIT
URL:	  https://github.com/ericwq/aprilsh
Group:	  System Environment/Daemons
%undefine _disable_source_fetch
Source0:  https://github.com/ericwq/aprilsh/releases/download/%{version}/aprilsh-%{version}.tar.gz
Source1:  apshd.initd
Source2:  apshd.logrotate
BuildRequires:	gcc
BuildRequires:	golang
BuildRequires:	ncurses-devel	
BuildRequires:	protobuf-compiler
Requires: ncurses-term
Requires: openssh-server

%description
Aprilsh: remote shell support intermittent or mobile network. inspired by mosh and zutty. 

* aprilsh is a remote shell based on UDP, port range 8100-8200.
* aprilsh server is apshd. run as daemon, support openrc.
* aprilsh client is apsh. run in a terminal window, suppor modern terminal control sequence.
* aprilsh depends openssh for user authentication.

%prep
%autosetup -n %{name}-%{version}
mkdir -p ~/go/{bin,pkg,src}

%build
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# build info required by go build
_git_tag=0.6.39
_git_commit=88adc2f0e1063e86f136c05af9744aed3f7bebf5
_git_branch=HEAD

_module_name=$(head ./go.mod | grep "^module" | awk '{print $2}')
_build_time=$(date)
_go_version=$(go version | grep "version" | awk '{print $3,$4}')

# install go dependencies
go mod tidy
# echo "current directory is $(pwd)"
echo "_builddir is %{_builddir}"

# compile protobuf code
protoc --go_out=. -I . ./protobufs/transportInstruction.proto
protoc --go_out=. -I . ./protobufs/hostInput.proto
protoc --go_out=. -I . ./protobufs/userInput.proto

echo "build server start: $(date)"
go build -ldflags="-s -w \
	-X ${_module_name}/frontend.GitTag=${_git_tag} \
	-X ${_module_name}/frontend.GoVersion=${_go_version} \
	-X ${_module_name}/frontend.GitCommit=${_git_commit} \
	-X ${_module_name}/frontend.GitBranch=${_git_branch} \
	-X ${_module_name}/frontend.BuildTime=${_build_time}" \
	-o %{_builddir}%{_bindir}/apshd ./frontend/server/*.go
echo "build server end	: $(date)"
echo "output server to	: %{_builddir}%{_bindir}/apshd"

echo "build client start: $(date)"
go build -ldflags="-s -w \
	-X $_module_name/frontend.GitTag=${_git_tag}\
	-X $_module_name/frontend.GoVersion=${_go_version}\
	-X $_module_name/frontend.GitCommit=${_git_commit}\
	-X $_module_name/frontend.GitBranch=${_git_branch}\
	-X $_module_name/frontend.BuildTime=${_build_time}"\
	-o "%{_builddir}%{_bindir}/apsh" ./frontend/client/*.go
echo "build client end	: $(date)"
echo "output client to	: %{_builddir}%{_bindir}/apsh"

# run unit test
go test ./encrypt/...
go test ./frontend/
APRILSH_APSHD_PATH="%{_builddir}%{_bindir}/apshd" go test ./frontend/server
go test ./frontend/client
go test ./network/...
go test ./statesync/...
go test ./terminal/...
go test ./util/...

%install
rm -rf $RPM_BUILD_ROOT
echo  %{buildroot}
install -Dm755 "%{_builddir}%{_bindir}/apshd" "%{buildroot}/%{_bindir}"
install -Dm755 "%{_builddir}%{_bindir}/apsh" "%{buildroot}/%{_bindir}"
install -Dm644 "${_sourcedir}/apshd.initd" "$i{buildroot}/etc/init.d/apshd"
install -Dm644 "${_sourcedir}/apshd.logrotate" "${buildroot}/etc/logrotate.d/apshd"
