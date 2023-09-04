#!/bin/bash
set -eux

hostname=$(hostname --fqdn)
if  [ "${hostname}" = 'moon-0' ]; then
remote_ip='10.2.0.22'
else
remote_ip='10.1.0.11'
fi

#
# kick the tires.
ping -n -c 4 $remote_ip || true
swanctl --list-conns
swanctl --list-sas
swanctl --stats
ip xfrm state
ip xfrm policy
