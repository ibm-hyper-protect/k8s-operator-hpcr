#!/bin/sh
set -eu

echo "Hello from busybox ref at $(date)" >> /srv/busybox/log.txt && cat /srv/busybox/log.txt