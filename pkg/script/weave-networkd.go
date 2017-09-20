package script

const WeaveNetworkdUnmaskPath = "/etc/systemd/network/50-weave.network"

const WeaveNetworkdUnmask = `[Match]
Name=weave datapath vethwe*

[Link]
Unmanaged=yes
`
