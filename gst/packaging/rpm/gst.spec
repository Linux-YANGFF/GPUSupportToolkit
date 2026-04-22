Name: gst
Version: 1.0.0
Release: 1%{?dist}
Summary: GPU Support Toolkit - GPU log analysis tool
License: MIT
URL: https://github.com/example/gst
BuildArch: x86_64
Requires: glibc, libGL, libX11

%description
GST is a tool for analyzing apitrace and profile logs.

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/var/lib/gst
cp gst-server %{buildroot}/usr/bin/
cp gst.desktop %{buildroot}/usr/share/applications/

%files
/usr/bin/gst-server
/usr/share/applications/gst.desktop
%dir /var/lib/gst

%post
mkdir -p /var/lib/gst 2>/dev/null || true

%postun
rmdir /var/lib/gst 2>/dev/null || true