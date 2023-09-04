#!/bin/bash
set -eux

domain=$(hostname --fqdn)

if  [ "$(hostname)" = 'moon-0' ]; then
local='moon-0'
local_ts='10.1.0.0/24'
remote='sun-0'
remote_ts='10.2.0.0/24'
remote_ip='10.2.0.22'
echo "init local $local $local_ts --> remote $remote $remote_ts $remote_ip"
echo "127.0.2.1 moon-0.vpn.gw.com moon-0" >> /etc/hosts
else
local='sun-0'
local_ts='10.2.0.0/24'
remote='moon-0'
remote_ts='10.1.0.0/24'
remote_ip='10.1.0.11'
echo "init local $local $local_ts --> remote $remote $remote_ts $remote_ip"
echo "127.0.2.1 sun-0.vpn.gw.com sun-0" >> /etc/hosts
fi

echo "192.168.7.11 moon-0.vpn.gw.com" >> /etc/hosts
echo "192.168.7.22 sun-0.vpn.gw.com" >> /etc/hosts

#
# install the strongswan charon daemon (has native systemd integration).
# NB do not install the strongswan package as it will use the legacy stuff (e.g. ipsec.conf).

# apt-get install -y charon-systemd
# systemctl status strongswan-swanctl
# swanctl --version


#
# configure.

cp /vagrant/shared/example-ca/example-ca-crt.pem /etc/swanctl/x509ca
cp /vagrant/shared/example-ca/$local.vpn.gw.com-crt.pem /etc/swanctl/x509
cp /vagrant/shared/example-ca/$local.vpn.gw.com-key.pem /etc/swanctl/private
mv /etc/swanctl/swanctl.conf ~/swanctl.conf.orig
# see https://wiki.strongswan.org/projects/strongswan/wiki/StrongswanConf
cat >/etc/swanctl/swanctl.conf <<EOF
connections {
    net-net {
        local {
            auth = pubkey
            certs = $local.vpn.gw.com-crt.pem
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

#
# kick the tires.

ping -n -c 4 $remote_ip || true
swanctl --list-conns
swanctl --list-sas
swanctl --stats
ip xfrm state
ip xfrm policy


#
# add useful commands to the shell history.

cat >>~/.bash_history <<EOF
ping -n $remote_ip
swanctl --list-conns
swanctl --list-sas
swanctl --stats
ip xfrm state
ip xfrm policy
#systemctl status strongswan-swanctl
#systemctl restart strongswan-swanctl
#journalctl -u charon-systemd
#journalctl -u strongswan-swanctl
EOF
