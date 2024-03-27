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
skalibs is a package centralizing the free software / open source C development files used for building all software at skarnet.org: it contains essentially general-purpose libraries. You will need to install skalibs if you plan to build skarnet.org software.

%package  devel
Summary:  Set of general-purpose C programming libraries for skarnet.org software. (development files)
Group:	  Development/Libraries
Requires: %{name} = %{version}-%{release}
%description devel
skalibs development files.

%package  static
Summary:  Set of general-purpose C programming libraries for skarnet.org software. (static library)
Group:	  Development/Libraries
Requires: %{name}-devel = %{version}-%{release}
%description static
skalibs static library

%prep
%autosetup -n %{name}-%{version}
sed -i "s|@@VERSION@@|%{version}|" -i %{SOURCE1}

%build
./configure --enable-shared --enable-static --libdir=%{_libdir}
make

%install
rm -rf %{buildroot}
make DESTDIR=%{buildroot} install
install -D -m 0644 "%{SOURCE1}" "%{buildroot}/%{_libdir}"

%files
%defattr(-,root,root,0755)
%{_libdir}/libskarnet.so.*

%files devel
%defattr(-,root,root,0755)
%{_includedir}/*
%{_libdir}/libskarnet.so.*
%{_libdir}/pkgconfig/skalibs.pc

%files static
%defattr(-,root,root,0755)
%{_libdir}/*.a

%changelog
* Thu Mar 28 2024 Wang Qi <ericwq057@qq.com> - v0.1
- First version being packaged
