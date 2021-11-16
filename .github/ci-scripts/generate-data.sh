#!/usr/bin/env bash

mkdir cache
filename="cache/$(uuidgen)"
dd if=/dev/urandom of="$filename" bs=1M count=16
md5sum "$filename" > checksum
