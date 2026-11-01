#!/bin/sh
set -eu
if ! /usr/sbin/nft list table inet mirrorouc_monitor >/dev/null 2>&1; then
    /usr/sbin/nft -f /etc/mirrorouc/protocol-counters.nft
fi
