#!/bin/bash
set -eux

CONF=/etc/swanctl/swanctl.conf
CONNECTION_YAML=connection.yaml
HOSTS=/etc/hosts

function init() {
    # init before pod running
    # prepare hosts.j2
    if [ ! -f hosts.j2 ]; then
        cp $HOSTS  hosts.j2
        cat templates_hosts.j2 >> hosts.j2
    fi
    # prepare swanctl.conf.j2
    if [ ! -f swanctl.conf.j2 ]; then
        mv $CONF swanctl.conf.orig
        cp templates_swanctl.conf.j2 swanctl.conf.j2
    fi
}

function refresh() {
    # refresh strongswan connections
    # format connection rules into connection.yaml
    printf "connection: \n" > $CONNECTION_YAML
    for rule in "$@"
    do
        arr=("${rule//,/ }")
        connectionName=${arr[0]}
        localCN=${arr[1]}
        localPublicIp=${arr[2]}
        localPrivateCidrs=${arr[3]}
        remoteCN=${arr[4]}
        remotePublicIp=${arr[5]}
        remotePrivateCidrs=${arr[6]}
        printf "  - name: %s\n" "${connectionName}" >> $CONNECTION_YAML
        printf "    localCN: %s\n" "${localCN}" >> $CONNECTION_YAML
        printf "    localPublicIp: %s\n" "${localPublicIp}" >> $CONNECTION_YAML
        printf "    localPrivateCidrs: %s\n" "${localPrivateCidrs}" >> $CONNECTION_YAML
        printf "    remoteCN: %s\n" "${remoteCN}" >> $CONNECTION_YAML
        printf "    remotePublicIp: %s\n" "${remotePublicIp}" >> $CONNECTION_YAML
        printf "    remotePrivateCidrs: %s\n" "${remotePrivateCidrs}" >> $CONNECTION_YAML
    done
    # use j2 to generate hosts and swanctl.conf
    j2 hosts.j2 $CONNECTION_YAML -o $HOSTS
    j2 swanctl.conf.j2 $CONNECTION_YAML -o $CONF

    # reload strongswan connections
    /usr/sbin/swanctl --load-all
}

rules=${*:2:${#}}
opt=$1
case $opt in
 init)
        echo "init ${rules}"
        init "${rules}"
        ;;
 refresh)
        echo "refresh ${rules}"
        refresh "${rules}"
        ;;
 *)
        echo "Usage: $0 [init|refresh]"
        exit 1
        ;;
esac