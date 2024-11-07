#!/bin/bash

# A script for ensuring our arguments have not drifted.

# # $ readelf -n /usr/lib/php/modules/compass.so
#
# Displaying notes found in: .note.gnu.build-id
# Owner                Data size 	Description
# GNU                  0x00000014	NT_GNU_BUILD_ID (unique build ID bitstring)
#   Build ID: f3292e5e81429fcc9d40f29eaaff2c4789aae17c
#
# Displaying notes found in: .note.stapsdt
#   Owner                Data size 	Description
# stapsdt              0x00000039	NT_STAPSDT (SystemTap probe descriptors)
#   Provider: compass
#   Name: request_shutdown
#   Location: 0x000000000000cd48, Base: 0x0000000000064517, Semaphore: 0x0000000000000000
#   Arguments: -8@%rdi
# stapsdt              0x0000004d	NT_STAPSDT (SystemTap probe descriptors)
#   Provider: compass
#   Name: php_function
#   Location: 0x000000000000e62a, Base: 0x0000000000064517, Semaphore: 0x0000000000000000
#   Arguments: -8@%rdi -8@%r14 -8@%r15 -8@%rbx

FILE=$1

# Validate request_shutdown args.
if readelf -n ${FILE} | grep -q 'Arguments: -8@%rdi'; then
  echo "request_shutdown args are correct"
else
  echo "request_shutdown args are incorrect. We found:"
  readelf -n ${FILE}
  exit 1
fi

# Validate php_function args.
if readelf -n ${FILE} | grep -q 'Arguments: -8@%rbx -8@%r14 -8@%rax -8@%r13 -8@%rbp'; then
  echo "php_function args are correct"
else
  echo "php_function args are incorrect. We found:"
  readelf -n ${FILE}
  exit 1
fi