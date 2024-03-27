Name:	  skalibs
Version:  2.14.1.1
Release:  1%{?dist}
Summary:  Set of general-purpose C programming libraries for skarnet.org software.
License:  ISC
URL:	  https://skarnet.org/software/skalibs/
Group:	  System Environment/Libraries
%undefine _disable_source_fetch
Source0:  https://skarnet.org/software/skalibs/%{name}-%{version}.tar.gz
Source1:  skalibs.pc

%description
skalibs is a package centralizing the free software / open source C development files used for building all software at skarnet.org: it contains essentially general-purpose libraries. You will need to install skalibs if you plan to build skarnet.org software. The point is that you won't have to download and compile big libraries, and care about portability issues, everytime you need to build a package: do it only once.

%prep
%autosetup -n %{name}-%{version}
sed -i "s|@@VERSION@@|$pkgver|" -i "$srcdir"/*.pc

%build
./configure --enable-shared --enable-static --libdir=/usr/lib
make

%install
make DESTDIR=%{buildroot} install
install -D -m 0644 "$srcdir/skalibs.pc" "$pkgdir/usr/lib/pkgconfig/skalibs.pc"
