#!/bin/sh
set -eu

echo "Hello from busybox at $(date)" >> /srv/busybox/log.txt && cat /srv/busybox/log.txt