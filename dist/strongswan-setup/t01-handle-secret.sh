#!/bin/bash
set -eux

ca_file_name='ipsec'
mkdir -p /vagrant/$ca_file_name
pushd /vagrant/$ca_file_name
hostname=$(hostname --fqdn)
ca_common_name="$hostname.vpn.gw.com"
#
# create the client certificates to authenticate into the vpn.
if [ ! -f $hostname-crt.pem ]; then
    # 创建私钥
    openssl genrsa \
        -out $hostname-key.pem \
        2048 \
        2>/dev/null
    chmod 400 $hostname-key.pem
    # 创建证书签名请求
    openssl req -new \
        -sha256 \
        -subj "/CN=$ca_common_name" \
        -key $hostname-key.pem \
        -out $hostname-csr.pem
    # 基于 root 证书，root 私钥，创建 ipsec 应用 证书
    openssl x509 -req -sha256 \
        -CA /etc/ipsec/x509ca/ca.crt \
        -CAkey /etc/ipsec/x509ca/tls.key \
        -CAcreateserial \
        -extensions a \
        -extfile <(echo "[a]
            extendedKeyUsage=critical,clientAuth,serverAuth
            ") \
        -days 365 \
        -in  $hostname-csr.pem \
        -out $hostname-crt.pem
fi
