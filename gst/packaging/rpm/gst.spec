Name: gst
Version: %{pkg_version}
Release: 1%{?dist}
Summary: GPU Support Toolkit - GPU log analysis tool
License: MIT
URL: https://github.com/example/gst
Source0: gst-%{pkg_version}.tar.gz
Requires: glibc
Recommends: xdg-utils

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
/usr/bin/gst-cli
/usr/bin/gst-ui
/usr/bin/gst-create-desktop-shortcut
/usr/share/applications/gst.desktop
/usr/share/icons/hicolor/scalable/apps/gst.svg
/usr/share/gst/web
%dir /var/lib/gst

%post
mkdir -p /var/lib/gst 2>/dev/null || true
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor >/dev/null 2>&1 || true
fi

%preun
if [ -f /var/run/gst-server.pid ]; then
    kill "$(cat /var/run/gst-server.pid)" 2>/dev/null || true
    rm -f /var/run/gst-server.pid
fi
if [ -f /run/gst-server.pid ]; then
    kill "$(cat /run/gst-server.pid)" 2>/dev/null || true
    rm -f /run/gst-server.pid
fi
pkill -f /usr/bin/gst-server 2>/dev/null || true

%postun
rm -f /var/run/gst-server.pid /run/gst-server.pid
if [ -d /var/lib/gst ] && ! find /var/lib/gst -mindepth 1 -print -quit | grep -q .; then
    rmdir /var/lib/gst 2>/dev/null || true
fi
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor >/dev/null 2>&1 || true
fi
