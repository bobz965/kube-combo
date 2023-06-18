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
cp /etc/ipsec/x509ca/ca.crt /etc/swanctl/x509ca
cp /vagrant/ipsec/$local-key.pem /etc/swanctl/private
cp /vagrant/ipsec/$local-crt.pem /etc/swanctl/x509

#
# configure /etc/swanctl/swanctl.conf.
mv /etc/swanctl/swanctl.conf ~/swanctl.conf.orig
# see https://wiki.strongswan.org/projects/strongswan/wiki/StrongswanConf
cat >/etc/swanctl/swanctl.conf <<EOF
connections {
    net-net {
        local {
            auth = pubkey
            certs = $local-crt.pem
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
