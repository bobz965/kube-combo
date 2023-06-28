#!/bin/bash
set -eux

ca_file_name='example-ca'
ca_common_name='Example CA'

mkdir -p /vagrant/shared/$ca_file_name
pushd /vagrant/shared/$ca_file_name

# create the CA certificate.
if [ ! -f $ca_file_name-crt.pem ]; then
    # 创建私钥
    openssl genrsa \
        -out $ca_file_name-key.pem \
        2048 \
        2>/dev/null
    chmod 400 $ca_file_name-key.pem
    # 创建证书签名请求
    openssl req -new \
        -sha256 \
        -subj "/CN=$ca_common_name" \
        -key $ca_file_name-key.pem \
        -out $ca_file_name-csr.pem
    # 创建证书
    openssl x509 -req -sha256 \
        -signkey $ca_file_name-key.pem \
        -extensions a \
        -extfile <(echo "[a]
            basicConstraints=critical,CA:TRUE,pathlen:0
            keyUsage=critical,digitalSignature,keyCertSign,cRLSign
            ") \
        -days 365 \
        -in  $ca_file_name-csr.pem \
        -out $ca_file_name-crt.pem

    # 将证书转换为der格式， 非必须
    # openssl x509 \
    #     -in $ca_file_name-crt.pem \
    #     -outform der \
    #     -out $ca_file_name-crt.der

    # dump the certificate contents (for logging purposes).
    # openssl x509 -noout -text -in $ca_file_name-crt.pem
fi

# create the client certificates to authenticate into the vpn.
vpn_client_common_names=(
    'moon-0.vpn.gw.com'
    'sun-0.vpn.gw.com'
)
for common_name in "${vpn_client_common_names[@]}"; do
    if [ ! -f $common_name-crt.pem ]; then
        # 创建私钥
        openssl genrsa \
            -out $common_name-key.pem \
            2048 \
            2>/dev/null
        chmod 400 $common_name-key.pem
        # 创建证书签名请求
        openssl req -new \
            -sha256 \
            -subj "/CN=$common_name" \
            -key $common_name-key.pem \
            -out $common_name-csr.pem
        # 基于 root 证书，root 私钥，创建 ipsec 应用 证书
        openssl x509 -req -sha256 \
            -CA $ca_file_name-crt.pem \
            -CAkey $ca_file_name-key.pem \
            -CAcreateserial \
            -extensions a \
            -extfile <(echo "[a]
                extendedKeyUsage=critical,clientAuth,serverAuth
                ") \
            -days 365 \
            -in  $common_name-csr.pem \
            -out $common_name-crt.pem

        # 将证书转换为 p12 格式， 非必须
        # openssl pkcs12 -export \
        #     -keyex \
        #     -inkey $common_name-key.pem \
        #     -in $common_name-crt.pem \
        #     -certfile $common_name-crt.pem \
        #     -passout pass: \
        #     -out $common_name-key.p12
        # dump the certificate contents (for logging purposes).

        # openssl x509 -noout -text -in $common_name-crt.pem
        # openssl pkcs12 -info -nodes -passin pass: -in $common_name-key.p12
    fi
done


# extendedKeyUsage 字段需要注意下，基于 cert-manager 维护的时候需要具备对应的 usage
# 这个脚本对 pem 文件有点泛用，需要基于命令查看下具体内容: https://stackoverflow.com/questions/63195304/difference-between-pem-crt-key-files