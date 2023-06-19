#!/bin/bash
set -eux

hostname=$(hostname --fqdn)
if  [ "$(hostname)" = 'moon-0' ]; then
local='moon-0'
local_ts='10.1.0.0/24'
remote='sun-0'
remote_ts='10.2.0.0/24'
remote_ip='10.2.0.22'
else
local='sun-0'
local_ts='10.2.0.0/24'
remote='moon-0'
remote_ts='10.1.0.0/24'
remote_ip='10.1.0.11'
fi

#
# configure ca.
cp /etc/ipsec/certs/ca.crt /etc/swanctl/x509ca
cp /etc/ipsec/certs/tls.key /etc/swanctl/private
cp /etc/ipsec/certs/tls.crt /etc/swanctl/x509

#
# configure /etc/swanctl/swanctl.conf.
mv /etc/swanctl/swanctl.conf ~/swanctl.conf.orig
# see https://wiki.strongswan.org/projects/strongswan/wiki/StrongswanConf
cat >/etc/swanctl/swanctl.conf <<EOF
connections {
    net-net {
        local {
            auth = pubkey
            certs = tls.crt
        }
        remote {
            auth = pubkey
            id = "CN=$remote.vpn.gw.com"
        }
        remote_addrs = $remote.vpn.gw.com
        children {
            net-net {
                local_ts = $local_ts
                remote_ts = $remote_ts
                dpd_action = restart
                start_action = trap
            }
        }
    }
}
EOF

swanctl --load-all
