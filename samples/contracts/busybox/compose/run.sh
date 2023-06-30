#!/bin/sh
set -eu

echo "Hello from busybox, machine-id: $(cat /var/lib/busybox/machine-id)!"
