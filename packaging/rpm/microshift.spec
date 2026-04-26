%global goipath github.com/ausil/microshift-2.0

Name:           microshift
Version:        0.1.0
Release:        1%{?dist}
Summary:        Lightweight Kubernetes for Edge

License:        Apache-2.0
URL:            https://%{goipath}
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21
BuildRequires:  systemd-rpm-macros

Requires:       kubernetes1.32
Requires:       etcd
Requires:       cri-o
Requires:       containernetworking-plugins
Recommends:     lvm2
Suggests:       nfs-utils

%description
MicroShift 2.0 is a lightweight, single-node Kubernetes distribution
designed for edge computing. It orchestrates Fedora-packaged Kubernetes
components as separate systemd services.

%prep
%setup -q

%build
make build VERSION=%{version} GOFLAGS=-mod=vendor

%install
make install DESTDIR=%{buildroot}

%post
%systemd_post microshift.service

%preun
%systemd_preun microshift.service
%systemd_preun microshift-etcd.service
%systemd_preun microshift-apiserver.service
%systemd_preun microshift-controller-manager.service
%systemd_preun microshift-scheduler.service
%systemd_preun microshift-kubelet.service
%systemd_preun microshift-kube-proxy.service

%postun
%systemd_postun_with_restart microshift.service

%files
%license LICENSE
%{_bindir}/microshift
%dir %{_sysconfdir}/microshift
%config(noreplace) %{_sysconfdir}/microshift/config.yaml
%{_unitdir}/microshift.service
%{_unitdir}/microshift-etcd.service
%{_unitdir}/microshift-apiserver.service
%{_unitdir}/microshift-controller-manager.service
%{_unitdir}/microshift-scheduler.service
%{_unitdir}/microshift-kubelet.service
%{_unitdir}/microshift-kube-proxy.service
%dir %{_datadir}/microshift
%{_datadir}/microshift/assets/

%changelog
* %(date "+%a %b %d %Y") MicroShift Authors <microshift@example.com> - 0.1.0-1
- Initial package
