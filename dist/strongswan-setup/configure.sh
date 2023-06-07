#!/bin/sh
# ipsec site-to-site vpn setup: https://github.com/strongswan/strongswan#site-to-site-case
# generate pems
# strongswanCert.pem should be the same between local and remote ipsec vpn gw

# /etc/swanctl/x509ca/strongswanCert.pem
# /etc/swanctl/x509/moonCert.pem
# /etc/swanctl/private/moonKey.pem

# 1. generate /etc/swanctl/x509ca/strongswanCert.pem
pki --gen --type ed25519 --outform pem > strongswanKey.pem

pki --self --ca --lifetime 3652 --in strongswanKey.pem \
           --dn "C=CH, O=strongSwan, CN=strongSwan Root CA" \
           --outform pem > strongswanCert.pem

# use env HOSTNAME as is HOSTNAME to replace moon
# 2. generate /etc/swanctl/private/moonKey.pem
pki --gen --type ed25519 --outform pem > ${HOSTNAME}Key.pem

# 3. generate /etc/swanctl/x509/moonCert.pem by moonKey.pem
pki --req --type priv --in "${HOSTNAME}Key.pem" \
          --dn "C=CH, O=strongswan, CN=moon.strongswan.org" \
          --san moon.strongswan.org --outform pem > ${HOSTNAME}Req.pem

pki --issue --cacert strongswanCert.pem --cakey strongswanKey.pem \
            --type pkcs10 --in ${HOSTNAME}Req.pem --serial 01 --lifetime 3652 \
            --outform pem > "${HOSTNAME}Cert.pem"

\cp strongswanCert.pem /etc/swanctl/x509ca/strongswanCert.pem
\cp ${HOSTNAME}Key.pem "/etc/swanctl/private/${HOSTNAME}Key.pem"
\cp "${HOSTNAME}Cert.pem" "/etc/swanctl/x509/${HOSTNAME}Cert.pem"


# LOCAL_KEY_PEM="/etc/swanctl/private/${HOSTNAME}Key.pem"
LOCAL_KEY_PEM="${HOSTNAME}Key.pem"

# LOCAL_CERT_PEM="/etc/swanctl/x509/${HOSTNAME}Cert.pem"
LOCAL_CERT_PEM="${HOSTNAME}Cert.pem"

# calculate local subnet cidr
cidr2net() {
    local i ip mask netOctets octets
    ip="${1%/*}"
    mask="${1#*/}"
    octets=$(echo "$ip" | tr '.' '\n')

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

# configure /etc/swanctl/swanctl.conf
cp -f /etc/swanctl/setup/swanctl.conf /etc/swanctl/swanctl.conf
sed 's||REMOTE_ADDRS'"${REMOTE_ADDRS}"'|' -i /etc/swanctl/swanctl.conf
sed 's|LOCAL_CERT_PEM|'"${LOCAL_CERT_PEM}"'|' -i /etc/swanctl/swanctl.conf
sed 's|LOCAL_TS|'"${LOCAL_TS}"'|' -i /etc/swanctl/swanctl.conf
sed 's|REMOTE_TS|'"${REMOTE_TS}"'|' -i /etc/swanctl/swanctl.conf

# load and start
swanctl --load-creds
swanctl --load-conns
