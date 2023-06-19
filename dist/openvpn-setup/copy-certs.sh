#!/bin/bash
set -eux
cp /etc/ovpn/certs/tls.key /etc/openvpn/certs/pki/private/server.key 
cp /etc/ovpn/certs/ca.crt /etc/openvpn/certs/pki/ca.crt 
cp /etc/ovpn/certs/tls.crt /etc/openvpn/certs/pki/issued/server.crt 
cp /etc/ovpn/dh/dh.pem /etc/openvpn/certs/pki/dh.pem

