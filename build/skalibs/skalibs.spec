#
# spec file for package skalibs
#
# Contributor: Wang Qi <ericwq057@qq.com>
#

%define debug_package %{nil}

Name:	  skalibs
Version:  2.14.1.1
Release:  1%{?dist}
Summary:  Set of general-purpose C programming libraries for skarnet.org software.
License:  ISC
URL:	  https://skarnet.org/software/%{name}
Group:	  System Environment/Libraries
%undefine _disable_source_fetch
Source0:  https://skarnet.org/software/%{name}/%{name}-%{version}.tar.gz
Source1:  skalibs.pc
BuildRequires: gcc make pkgconfig
%description
skalibs is a package centralizing the free software / open source C development files used for building all software at skarnet.org: it contains essentially general-purpose libraries. You will need to install skalibs if you plan to build skarnet.org software.

%package  devel
Summary:  Set of general-purpose C programming libraries for skarnet.org software. (development files)
Group:	  Development/Libraries
Requires: %{name} = %{version}-%{release}
Requires: pkgconfig
%description devel
skalibs is a package centralizing the free software / open source C development files used for building all software at skarnet.org: it contains essentially general-purpose libraries. You will need to install skalibs if you plan to build skarnet.org software.

This subpackage holds the development headers and sysdeps files for the library.

%package  static
Summary:  Set of general-purpose C programming libraries for skarnet.org software. (static library)
Group:	  Development/Libraries
%description static
skalibs is a package centralizing the free software / open source C development files used for building all software at skarnet.org: it contains essentially general-purpose libraries. You will need to install skalibs if you plan to build skarnet.org software.

This subpackage contains the static version of the library used for development.

%prep
%autosetup -n %{name}-%{version}
sed -i "s|@@VERSION@@|%{version}|" -i %{SOURCE1}
cat %{SOURCE1}

%build
./configure --enable-shared --enable-static --libdir=%{_libdir} --dynlibdir=%{_libdir} \
	--with-pkg-config-libdir=%{_libdir}/pkgconfig \
	--sysdepdir=%{_libdir}/skalibs/sysdeps
make %{?_smp_mflags}

%install
rm -rf %{buildroot}
make install DESTDIR=%{buildroot}

# copy pkgconfig
install -D -m 0644 "%{SOURCE1}" "%{buildroot}%{_libdir}/pkgconfig/skalibs.pc"

# copy doc
mkdir -p %{buildroot}%{_docdir}/%{name}
cp -r "doc/" "%{buildroot}%{_docdir}/%{name}/"

%files
%defattr(-,root,root,0755)
%{_libdir}/libskarnet.so.*
%{_libdir}/libskarnet.so

%files devel
%{_docdir}/%{name}/*
%defattr(-,root,root,0755)
%{_includedir}/skalibs/*
%{_libdir}/skalibs/sysdeps
%{_libdir}/pkgconfig/skalibs.pc

%files static
%defattr(-,root,root,0755)
%{_libdir}/libskarnet.a

%changelog
* Thu Mar 28 2024 Wang Qi <ericwq057@qq.com> - v0.1
- First version being packaged
