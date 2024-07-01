Compass
=======

A tool for pointing developers in the right direction for performance issues.

<img src="/logo.png" width="100">

[![📋 Extension: Test](https://github.com/skpr/compass/actions/workflows/extension_test.yml/badge.svg)](https://github.com/skpr/compass/actions/workflows/extension_test.yml) [![📋 Collector: Test](https://github.com/skpr/compass/actions/workflows/collector_test.yml/badge.svg)](https://github.com/skpr/compass/actions/workflows/collector_test.yml)

----

## Architecture

```mermaid
flowchart LR
   Extension[<b>PECL Extension</b>\n<i>Rust</i>] --> compass_fpm_request_init[<b>compass_fpm_request_init</b>\n<i>Probe</i>]
   Extension --> compass_fpm_request_shutdown[<b>compass_fpm_request_shutdown</b>\n<i>Probe</i>]
   Extension --> compass_php_function_begin[<b>compass_php_function_begin</b>\n<i>Probe</i>]
   Extension --> compass_php_function_end[<b>compass_php_function_end</b>\n<i>Probe</i>]

   compass_fpm_request_init --> eBPF[<b>eBPF Program</b>\n<i>CO-RE</i>]
   compass_fpm_request_shutdown --> eBPF
   compass_php_function_begin --> eBPF
   compass_php_function_end --> eBPF

   eBPF --> function_end[<b>function_end</b>\n<i>Ring Buffer</i>]
   eBPF --> request_shutdown[<b>request_shutdown</b>\n<i>Ring Buffer</i>]

   function_end --> Collector[<b>Collector</b>\n<i>Go</i>]
   request_shutdown --> Collector

   Collector --> Stdout[<b>Stdout</b>\n<i>Go Plugin</i>]
```

## Components

| Directory | Description                                                                                      |
|-----------|--------------------------------------------------------------------------------------------------|
| extension | PHP extension which implements USDT probes using PHP's Oberserver APi.                           |
| bpftrace  | bpftrace scripts for testing the extension and demonstrating how the probes can be utilised.     |
| example   | Example for testing purposes.                                                                    |
| collector | Listens to USDT probes, collates them and sends them to the collector plugin (stdout, file etc). |

