#/usr/bin/bash

cp -r /original-files/* /var/lib/sss/
/usr/sbin/sssd -i
