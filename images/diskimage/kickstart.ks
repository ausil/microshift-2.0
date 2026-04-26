# MicroShift bootc anaconda-iso kickstart
#
# Used by bootc-image-builder when building anaconda-iso images.
# Embed in config.toml via [customizations.installer.kickstart].

network --bootproto=dhcp --activate
clearpart --all --initlabel --disklabel=gpt
reqpart --add-boot
part / --grow --fstype xfs
part /var/lib/microshift --size=5120 --fstype xfs
firewall --enabled --port=6443:tcp,22:tcp,10250:tcp,2379:tcp
services --enabled=sshd,crio,microshift
user --name=microshift --groups=wheel
rootpw --iscrypted --lock locked
reboot
