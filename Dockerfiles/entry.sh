#!/bin/bash

echo "===ARGS==="
echo "$@"
echo "=========="


/usr/bin/apisix init
/usr/local/openresty/bin/openresty -p /usr/local/apisix -g 'daemon off;' # remove /usr/bin/apisix init_etcd
