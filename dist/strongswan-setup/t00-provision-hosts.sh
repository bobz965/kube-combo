#!/bin/bash
set -eux

hostname=$(hostname --fqdn)
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
