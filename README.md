## Toolchain

Compass leverages a powerful ecosystem of tools for development, building, and observability:

```mermaid
graph TD
    subgraph Development
        Mise[Mise] --> Go[Go]
        Mise --> BPF2Go[bpf2go]
        Mise --> Lint[golangci-lint]
    end

    subgraph Build_Runtime
        Docker[Docker] --> Alpine[Alpine Linux]
        Alpine --> Clang[Clang/LLVM]
        Alpine --> LibBpf[libbpf]
    end

    subgraph Observability
        Compass[Compass] --> OpenTelemetry[OpenTelemetry]
        Compass --> Jaeger[Jaeger]
        Compass --> eBPF[eBPF]
    end
