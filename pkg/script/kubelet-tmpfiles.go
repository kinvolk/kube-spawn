package script

const KubeletTmpfilesPath = "/etc/tmpfiles.d/kubelet.conf"

const KubeletTmpfiles = `d /var/lib/kubelet 0755 - - -`
