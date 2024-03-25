Name:		aprilsh
Version:	0.6.39
Release:	1%{?dist}
Summary:	Remote shell support intermittent or mobile network

License:	MIT
URL:		https://github.com/ericwq/aprilsh
Source0:	https://github.com/ericwq/aprilsh/releases/download/%{version}/aprilsh-%{version}.tar.gz

BuildRequires:  gcc
BuildRequires:  golang
BuildRequires:	ncurses-devel	
BuildRequires:	protobuf-compiler


%description
Aprilsh: remote shell support intermittent or mobile network. inspired by mosh and zutty. 
aprilsh is a remote shell based on UDP, authenticate user via ssh.

%prep
%setup -q
