#!/bin/bash
set -euo pipefail
# generate client cert based given client key name and service ip
# $1 should be client key name
# $2 should be lb service external ip or floating ip

EASY_RSA_LOC="/etc/openvpn/certs"
cd $EASY_RSA_LOC
MY_IP_ADDR="$2"
./easyrsa build-client-full $1 nopass
cat >${EASY_RSA_LOC}/pki/$1.ovpn <<EOF
client
nobind
dev tun
remote-cert-tls server # mitigate mitm
remote ${MY_IP_ADDR} 1194 udp  
# default udp 1194
# defualt tcp 443
redirect-gateway def1
<key>
`cat ${EASY_RSA_LOC}/pki/private/$1.key`
</key>
<cert>
`cat ${EASY_RSA_LOC}/pki/issued/$1.crt`
</cert>
<ca>
`cat ${EASY_RSA_LOC}/pki/ca.crt`
</ca>
EOF
cat pki/$1.ovpn
