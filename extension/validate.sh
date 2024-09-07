#!/bin/bash

# A script for ensuring our arguments have not drifted.

# # $ readelf -n /usr/lib/php/modules/compass.so
#
# Displaying notes found in: .note.gnu.build-id
#   Owner                Data size 	Description
#   GNU                  0x00000014	NT_GNU_BUILD_ID (unique build ID bitstring)
#     Build ID: cb4d92781fc5c3a8b8db668910840b04762e4104
# 
# Displaying notes found in: .note.stapsdt
#   Owner                Data size 	Description
#   stapsdt              0x00000045	NT_STAPSDT (SystemTap probe descriptors)
#     Provider: compass
#     Name: php_function
#     Location: 0x000000000000fa11, Base: 0x000000000005e9eb, Semaphore: 0x0000000000000000
#     Arguments: -8@%rdi -8@%rbx -8@%r14
#   stapsdt              0x00000039	NT_STAPSDT (SystemTap probe descriptors)
#     Provider: compass
#     Name: request_shutdown
#     Location: 0x000000000000fe15, Base: 0x000000000005e9eb, Semaphore: 0x0000000000000000
#     Arguments: -8@%rdi

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
if readelf -n ${FILE} | grep -q 'Arguments: -8@%rdi -8@%rbx -8@%r14'; then
  echo "php_function args are correct"
else
  echo "php_function args are incorrect. We found:"
  readelf -n ${FILE}
  exit 1
fi