#!/bin/bash
set -euo pipefail

CONF=/etc/swanctl/swanctl.conf
CONNECTION_YAML=connection.yaml
HOSTS=/etc/hosts
TEMPLATE_HOSTS=template-hosts.j2
TEMPLATE_SWANCTL_CONF=template-swanctl.conf.j2
TEMPLATE_CHECK=template-check.j2
CHECK_SCRIPT=check

function init() {
    # prepare hosts.j2
    if [ ! -f hosts.j2 ]; then
        cp $HOSTS  hosts.j2
        cat $TEMPLATE_HOSTS >> hosts.j2
    fi
    # prepare swanctl.conf.j2
    if [ ! -f swanctl.conf.j2 ]; then
        mv $CONF swanctl.conf.orig
        cp $TEMPLATE_SWANCTL_CONF swanctl.conf.j2
    fi

    #
    # configure ca
    cp /etc/ipsec/certs/ca.crt /etc/swanctl/x509ca
    cp /etc/ipsec/certs/tls.key /etc/swanctl/private
    cp /etc/ipsec/certs/tls.crt /etc/swanctl/x509

}

function refresh() {
    # 1. init
    init
    # 2. refresh connections
    # format connections into connection.yaml
    printf "connections: \n" > $CONNECTION_YAML
    IFS=',' read -r -a array <<< "${connections}"
    for connection in "${array[@]}"
    do
        # echo "show connection: ${connection}"
        IFS=' ' read -r -a conn <<< "${connection}"
        name=${conn[0]}
        localCN=${conn[1]}
        localPublicIp=${conn[2]}
        localPrivateCidrs=${conn[3]}
        remoteCN=${conn[4]}
        remotePublicIp=${conn[5]}
        remotePrivateCidrs=${conn[6]}
        { 
        printf "  - name: %s\n" "${name}"
        printf "    localCN: %s\n" "${localCN}"
        printf "    localPublicIp: %s\n" "${localPublicIp}"
        printf "    localPrivateCidrs: %s\n" "${localPrivateCidrs}"
        printf "    remoteCN: %s\n" "${remoteCN}"
        printf "    remotePublicIp: %s\n" "${remotePublicIp}"
        printf "    remotePrivateCidrs: %s\n" "${remotePrivateCidrs}"
        } >> $CONNECTION_YAML
    done
    # 3. generate hosts and swanctl.conf
    # use j2 to generate hosts and swanctl.conf
    j2 hosts.j2 $CONNECTION_YAML -o $HOSTS
    j2 swanctl.conf.j2 $CONNECTION_YAML -o $CONF
    j2 $TEMPLATE_CHECK $CONNECTION_YAML -o $CHECK_SCRIPT
    chmod +x $CHECK_SCRIPT

    # 4. reload strongswan connections
    /usr/sbin/swanctl --load-all
}

if [ $# -eq 0 ]; then
    echo "Usage: $0 [init|refresh]"
    exit 1
fi
connections=${*:2:${#}}
opt=$1
case $opt in
 init)
        init
        ;;
 refresh)
        refresh "${connections}"
        ;;
 *)
        echo "Usage: $0 [init|refresh]"
        exit 1
        ;;
esac