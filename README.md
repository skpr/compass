Compass
=======

A tool designed to guide developers in the right direction for identifying and resolving performance issues.

<img src="/logo/compass.png" width="100">

----

## Architecture

```mermaid
flowchart LR
   Extension["PECL Extension (Rust)"] --> compass_php_function["compass_php_function (USDT)"]
   Extension --> compass_request_shutdown["compass_request_shutdown (USDT)"]

   compass_php_function --> eBPF[eBPF Program]
   compass_request_shutdown --> eBPF

   eBPF --> Events["Events (Ring Buffer)"]

   Events --> CLI["CLI (Go)"]
   Events --> Sidecar["Sidecar (Go)"]

   Sidecar --> Stdout
   Sidecar --> OpenTelemetry
```

## Performance

<img src="/docs/performance.png">

Performance data can be found in Github Actions for [this build](https://github.com/skpr/compass/pull/77).

## Components

| Directory | Description                                                                                  |
|-----------|----------------------------------------------------------------------------------------------|
| extension | PHP extension which implements USDT probes.                                                  |
| cli       | Commandline interface that collects telemetry from USDT probes.                              |
| sidecar   | Sidecar that collects telemetry from USDT probes and log to stdout.                          |

## CLI

<img src="/docs/cli.png">

## Trace

Compass provides developers with 2 types of traces:

* Complete - All function calls with start and end times.
* Segmented - Functions calls broken into segments. Used to demonstrate scale eg. CLI trace view.
* Count - Totals number of times a function was called.

Below is a condensed example for a complete trace:

```json
{
  "requestID": "55eefc9aa6008d539ef954aff41806a7",
  "startTime": 1726972907007464,
  "executionTime": 6054,
  "functionCalls": [
    {
      "name": "Symfony\\Component\\DependencyInjection\\Compiler\\Compiler::compile",
      "startTime": 1726972907128013,
      "endTime": 1726972907517795
    },
    {
      "name": "Symfony\\Component\\DependencyInjection\\ContainerBuilder::compile",
      "startTime": 1726972907128009,
      "endTime": 1726972907518593
    },
    {
      "name": "Drupal\\Core\\DrupalKernel::compileContainer",
      "startTime": 1726972907009684,
      "endTime": 1726972907518625
    },
    {
      "name": "Drupal\\Core\\DrupalKernel::initializeContainer",
      "startTime": 1726972907008223,
      "endTime": 1726972907612239
    },
    {
      "name": "Drupal\\Core\\DrupalKernel::boot",
      "startTime": 1726972907008040,
      "endTime": 1726972907612295
    },
  ],
}
```

## Images

**PHP Extension**

```
ghcr.io/skpr/compass:extension-8.3-latest
ghcr.io/skpr/compass:extension-8.2-latest
ghcr.io/skpr/compass:extension-8.1-latest
```

**Collector**

```
ghcr.io/skpr/compass:collector-latest
```

## How to test

```bash
git clone git@github.com:skpr/compass
cd compass

# Build the project and up a stack
make up

# Run the CLI
docker compose exec compass compass

# Go to http://localhost:8080
# Check back in the CLI
```

## Configuration

| COMPONENT     | ENVIRONMENT VARIABLE               | DEFAULT VALUE                   | Description                                                                                                                                                                     |
|---------------|------------------------------------|---------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Extension     | COMPASS_ENABLED                    | false                           | Enable the Compass extension                                                                                                                                                    |
| Extension     | COMPASS_FUNCTION_THRESHOLD         | 10000                           | Watermark for which functions to trace.                                                                                                                                         |
| CLI + Sidecar | COMPASS_PROCESS_NAME               | php-fpm                         | Name of the process to trace.                                                                                                                                                   |
| CLI + Sidecar | COMPASS_EXTENSION_PATH             | /usr/lib/php/modules/compass.so | Path to extension library which has probes.                                                                                                                                     |
| Sidecar       | COMPASS_SIDECAR_REQUEST_THRESHOLD  | 100                             | Watermark for which requests to trace.                                                                                                                                          |
| Sidecar       | COMPASS_SIDECAR_FUNCTION_THRESHOLD | 10                              | Watermark for which functions to trace.                                                                                                                                         |
| Sidecar       | COMPASS_SIDECAR_LOG_LEVEL          | info                            | Logging level for the collector component. Set to "debug" for debug notices.                                                                                                    |
| Sidecar       | COMPASS_SIDECAR_SINK               | stdout                          | Choose which metrics sink to use.                                                                                                                                               |
| Sidecar       | COMPASS_SIDECAR_OTEL_ENDPOINT      | http://jaeger:4318/v1/traces    | Endpoint to send OpenTelemetry traces to.                                                                                                                                       |
| Sidecar       | COMPASS_SIDECAR_OTEL_SERVICE_NAME  |                                 | Name of the service assign this trace to.                                                                                                                                       |
