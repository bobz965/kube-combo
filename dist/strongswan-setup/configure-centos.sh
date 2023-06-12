#!/bin/sh
set -euo pipefail
# ipsec site-to-site vpn setup: https://github.com/strongswan/strongswan#site-to-site-case
# strongswan pki gen ref: https://www.digitalocean.com/community/tutorials/how-to-set-up-an-ikev2-vpn-server-with-strongswan-on-ubuntu-22-04
# generate pems

# strongswanCert.pem should be the same between local and remote ipsec vpn gw in site-to-site case
# 1. generate root private key cert
MY_CN="${HOSTNAME}.strongswan.org"
if [ ! -e /ipsec.d/private/strongswanKey.pem ]
then
  echo "generate new generate root private key cert"
  strongswan pki --gen --type ed25519 --outform pem > /ipsec.d/private/strongswanKey.pem

  strongswan pki --self --ca --lifetime 3652 --in /ipsec.d/private/strongswanKey.pem \
           --dn "C=CH, O=strongSwan, CN=strongSwan Root CA" \
           --outform pem > /ipsec.d/cacerts/strongswanCert.pem
fi

# use env HOSTNAME as is HOSTNAME to replace moon
# 2. generate vpn server private key cert
strongswan pki --gen --type ed25519 --outform pem > /ipsec.d/private/${HOSTNAME}Key.pem

strongswan pki --req --type priv --in /ipsec.d/private/${HOSTNAME}Key.pem \
          --dn "C=CH, O=strongswan, CN=${MY_CN}" \
          --san ${MY_CN} --outform pem > /ipsec.d/private/${HOSTNAME}Req.pem

strongswan pki --issue --cacert /ipsec.d/cacerts/strongswanCert.pem --cakey /ipsec.d/private/strongswanKey.pem \
            --type pkcs10 --in /ipsec.d/private/${HOSTNAME}Req.pem --serial 01 --lifetime 3652 \
            --outform pem > /ipsec.d/certs/${HOSTNAME}Cert.pem


\cp -r /ipsec.d/* /etc/strongswan/ipsec.d/
# x509 mode need this copy
\cp /ipsec.d/certs/${HOSTNAME}Cert.pem /etc/strongswan/swanctl/x509/

LOCAL_KEY_PEM="${HOSTNAME}Key.pem"
LOCAL_CERT_PEM="${HOSTNAME}Cert.pem"

# calculate local subnet cidr
cidr2net() {
    ip="${1%/*}"
    mask="${1#*/}"
    octets=$(echo "$ip" | tr '.' '\n')
    i=0
    netOctets=""
    for octet in $octets; do
        i=$((i+1))
        if [ $i -le $(( mask / 8)) ]; then
            netOctets="$netOctets.$octet"
        elif [ $i -eq  $(( mask / 8 +1 )) ]; then
            netOctets="$netOctets.$((((octet / ((256 / ((2**((mask % 8)))))))) * ((256 / ((2**((mask % 8))))))))"
        else
            netOctets="$netOctets.0"
        fi
    done

    echo ${netOctets#.}
}

intAndIP="$(ip route get 8.8.8.8 | awk '/8.8.8.8/ {print $5 "-" $7}')"
int="${intAndIP%-*}"
ip="${intAndIP#*-}"
cidr="$(ip addr show dev "$int" | awk -vip="$ip" '($2 ~ ip) {print $2}')"

NETWORK="$(cidr2net $cidr)"
NETMASK="${cidr#*/}"
LOCAL_SUBNET_CIDR="${NETWORK}/${NETMASK}"

# configure swanctl.conf
MY_SWANCTL_CONF="/etc/strongswan/swanctl/conf.d/${HOSTNAME}swanctl.conf"
\cp swanctl.conf operator-swansctl.conf
sed 's|LOCAL_ADDRS|'"${PUBLIC_IP}"'|' -i operator-swansctl.conf
sed 's|REMOTE_ADDRS|'"${IPSEC_REMOTE_ADDRS}"'|' -i operator-swansctl.conf
sed 's|REMOTE_TS|'"${IPSEC_REMOTE_TS}"'|' -i operator-swansctl.conf
sed 's|LOCAL_CERT_PEM|'"${LOCAL_CERT_PEM}"'|' -i operator-swansctl.conf
sed 's|LOCAL_TS|'"${LOCAL_SUBNET_CIDR}"'|' -i operator-swansctl.conf
sed 's|HOSTNAME.strongswan.org|'"${MY_CN}"'|' -i operator-swansctl.conf

\cp operator-swansctl.conf ${MY_SWANCTL_CONF}
\cp strongswan.conf /etc/strongswan/strongswan.conf

# new version 5.9.10 load and start
swanctl --load-creds
swanctl --load-conns
