bpftrace
========

bpftrace is a high-level tracing language for Linux.

https://github.com/bpftrace/bpftrace

The bpftrace scripts in this directory as intended for testing the PHP extension component.

**Run the bpftrace script**

```bash
sed "s~FILE~$(../collector/compass-find-lib --process-name=php-fpm)~g" compass.bt | sudo bpftrace -
```
