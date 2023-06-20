#!/bin/bash
set -euo pipefail
# generate client cert based given client key name and service ip
# $1 should be client key name
# PUBLIC_IP should be lb service external ip or floating ip

EASY_RSA_LOC="/etc/openvpn/certs"
cd $EASY_RSA_LOC
client_key_name=$1
./easyrsa build-client-full "${client_key_name}" nopass
cat >${EASY_RSA_LOC}/pki/"${client_key_name}" <<EOF
client
nobind
dev tun
remote-cert-tls server # mitigate mitm
remote ${PUBLIC_IP} 1194 udp  
# default udp 1194
# defualt tcp 443
redirect-gateway def1
<key>
$(cat ${EASY_RSA_LOC}/pki/private/"${client_key_name}".key)
</key>
<cert>
$(cat ${EASY_RSA_LOC}/pki/issued/"${client_key_name}".crt)
</cert>
<ca>
$(cat ${EASY_RSA_LOC}/pki/ca.crt)
</ca>
EOF
cat pki/"${client_key_name}".ovpn
