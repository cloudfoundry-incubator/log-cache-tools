#!/bin/sh

set +eux

SCRIPTS_PATH="$(cd "$(dirname "$0")" ; pwd -P )"
WORKSPACE="$SCRIPTS_PATH/.."
CERTS_PATH="$WORKSPACE/log-cache-forwarders/pkg/egress/syslog/config/test-certs/"

cd $SCRIPTS_PATH
openssl genrsa -out $CERTS_PATH/rootCA.key 4096
openssl req -x509 -new -nodes -key $CERTS_PATH/rootCA.key -sha256 -days 1024 -subj "/C=US/ST=CA/O=MyOrg, Inc./CN=fakeca" -out $CERTS_PATH/rootCA.crt

openssl genrsa -out $CERTS_PATH/client.key 2048
openssl req -new -sha256 -key $CERTS_PATH/client.key -subj "/C=US/ST=CA/O=MyOrg, Inc./CN=fakecommonname" -out $CERTS_PATH/client.csr
openssl x509 -req -in $CERTS_PATH/client.csr -CA $CERTS_PATH/rootCA.crt -CAkey $CERTS_PATH/rootCA.key -CAcreateserial -CAserial $CERTS_PATH/rootCA.srl -out $CERTS_PATH/client.crt -days 365 -sha256

rm $CERTS_PATH/client.csr
rm $CERTS_PATH/rootCA.srl
