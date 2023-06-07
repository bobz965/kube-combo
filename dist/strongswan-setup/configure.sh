#!/bin/sh

# generate pems
# strongswanCert.pem should be the same between local and remote ipsec vpn gw

# /etc/swanctl/x509ca/strongswanCert.pem
# /etc/swanctl/private/moonKey.pem
# /etc/swanctl/x509/moonCert.pem

# generate /etc/swanctl/x509ca/strongswanCert.pem
pki --gen --type ed25519 --outform pem > strongswanKey.pem
pki --self --ca --lifetime 3652 --in strongswanKey.pem \
           --dn "C=CH, O=strongSwan, CN=strongSwan Root CA" \
           --outform pem > strongswanCert.pem

# use env POD_NAME to replace moon
# generate /etc/swanctl/private/moonKey.pem
pki --gen --type ed25519 --outform pem > ${POD_NAME}Key.pem

# generate /etc/swanctl/x509/moonCert.pem
pki --req --type priv --in "${POD_NAME}Key.pem" \
          --dn "C=CH, O=strongswan, CN=moon.strongswan.org" \
          --san moon.strongswan.org --outform pem > ${POD_NAME}Req.pem

pki --issue --cacert strongswanCert.pem --cakey /etc/swanctl/strongswanKey.pem \
            --type pkcs10 --in ${POD_NAME}Req.pem --serial 01 --lifetime 3652 \
            --outform pem > "${POD_NAME}Cert.pem"

\cp strongswanCert.pem /etc/swanctl/x509ca/strongswanCert.pem

LOCAL_KEY_PEM="/etc/swanctl/private/${POD_NAME}Key.pem"
\cp ${POD_NAME}Key.pem "${LOCAL_KEY_PEM}"

LOCAL_CERT_PEM="/etc/swanctl/x509/${POD_NAME}Cert.pem"
\cp "${POD_NAME}Cert.pem" "${LOCAL_CERT_PEM}"




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
