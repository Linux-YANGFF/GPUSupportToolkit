Name: gst
Version: %{pkg_version}
Release: 1%{?dist}
Summary: GPU Support Toolkit - GPU log analysis tool
License: MIT
URL: https://github.com/example/gst
Source0: gst-%{pkg_version}.tar.gz
Requires: glibc

%description
GST is a tool for analyzing apitrace, profile, and rawtrace logs.

%prep
%setup -q -c -T
tar -xzf %{SOURCE0}

%install
mkdir -p %{buildroot}
cp -a . %{buildroot}/
rm -rf %{buildroot}/DEBIAN

%files
/usr/bin/gst-server
/usr/bin/gst
/usr/share/applications/gst.desktop
/usr/share/gst/web
%dir /var/lib/gst

%post
mkdir -p /var/lib/gst 2>/dev/null || true

%preun
if [ -f /var/run/gst-server.pid ]; then
    kill "$(cat /var/run/gst-server.pid)" 2>/dev/null || true
    rm -f /var/run/gst-server.pid
fi
