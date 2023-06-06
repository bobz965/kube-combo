#!/bin/sh

# generate these pems
## strongswanCert.pem should be the same between local and remote ipsec vpn gw
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

local_key_pem="/etc/swanctl/private/${POD_NAME}Key.pem"
\cp ${POD_NAME}Key.pem "${local_key_pem}"

local_cert_pem="/etc/swanctl/x509/${POD_NAME}Cert.pem"
\cp "${POD_NAME}Cert.pem" "${local_cert_pem}"


# configure /etc/swanctl/swanctl.conf
cp -f /etc/swanctl/setup/swanctl.conf /etc/swanctl/swanctl.conf
sed 's||REMOTE_IPSEC_VPN_GW_IP'"${REMOTE_IPSEC_VPN_GW_IP}"'|' -i /etc/swanctl/swanctl.conf
sed 's|LOCAL_CERT_PEM|'"${LOCAL_CERT_PEM}"'|' -i /etc/swanctl/swanctl.conf
sed 's|LOCAL_SUBNET_CIDR|'"${LOCAL_SUBNET_CIDR}"'|' -i /etc/swanctl/swanctl.conf
sed 's|REMOTE_SUBNET_CIDR|'"${REMOTE_SUBNET_CIDR}"'|' -i /etc/swanctl/swanctl.conf



# load and start
swanctl --load-creds
swanctl --load-conns

