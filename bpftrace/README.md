bpftrace
========

bpftrace is a high-level tracing language for Linux.

https://github.com/bpftrace/bpftrace

The bpftrace scripts in this directory as intended for testing the PHP extension component.

## How to use

This section assumes you have a Docker container running with the Compass PHP extension installed.

**Identify the PID of the Docker container**

```bash
# grep
$ docker inspect b7827db3e8c7 | grep Pid

# jq
$ docker inspect b7827db3e8c7 | jq '.[] | .State.Pid' 
```

**Run the bpftrace script**

Update the `PID` references in the script to use the Pid identified in the step prior.

Run the script using:

```bash
# Print all FPM requests.
sudo bpftrace bpftrace/fpm_requests.bt

# Print all function calls.
sudo bpftrace bpftrace/fpm_functions.bt
```
