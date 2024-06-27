Compass
=======

A tool for pointing developers in the right direction for performance issues.

<img src="/logo.png" width="100">

----

## Architecture

```mermaid
flowchart LR
   Extension[<b>PECL Extension</b>\n<i>Rust</i>] --> skpr_fpm_request_init[<b>skpr_fpm_request_init</b>\n<i>Probe</i>]
   Extension --> skpr_fpm_request_shutdown[<b>skpr_fpm_request_shutdown</b>\n<i>Probe</i>]
   Extension --> skpr_php_function_begin[<b>skpr_php_function_begin</b>\n<i>Probe</i>]
   Extension --> skpr_php_function_end[<b>skpr_php_function_end</b>\n<i>Probe</i>]

   skpr_fpm_request_init --> eBPF[<b>eBPF Program</b>\n<i>CO-RE</i>]
   skpr_fpm_request_shutdown --> eBPF
   skpr_php_function_begin --> eBPF
   skpr_php_function_end --> eBPF

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

