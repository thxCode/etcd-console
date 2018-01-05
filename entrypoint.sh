#!/usr/bin/dumb-init /bin/bash
set -e

# start nginx
/usr/sbin/nginx -c /etc/nginx/nginx.conf

# start etcd-console
/usr/sbin/etcd-console $@
