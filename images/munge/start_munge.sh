#!/usr/bin/bash

echo "Set permissions for shared /run/munge"
chmod 755 /run/munge # It changes permissions of this shared directory in other containers as well

/usr/sbin/munged --foreground
