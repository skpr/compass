#!/bin/sh

sed "s~FILE~$(compass-find-lib --process-name=php-fpm)~g" /compass.bt | bpftrace -
