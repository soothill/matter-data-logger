# 1. Use Grandcentrix Zeroconf Library

- **Status**: Accepted
- **Date**: 2025-11-12

## Context and Problem Statement

The Matter Data Logger requires a mechanism to discover Matter devices on the local network. The primary discovery method for Matter devices is mDNS (Multicast DNS), which allows for zero-configuration service discovery. The application needs a robust, well-maintained Go library to handle mDNS browsing and service resolution.

## Decision Drivers

- **Correctness**: The library must correctly implement the mDNS and DNS-SD protocols as specified in RFC 6762 and RFC 6763.
- **API Simplicity**: The library should provide a clean, easy-to-use API for browsing services, resolving them, and handling updates.
- **Maintenance**: The library should be actively maintained to address bugs and adapt to new Go versions.
- **Dependencies**: The library should have minimal external dependencies to keep the project lean.
- **Cross-Platform Support**: The library must work reliably on Linux, macOS, and Windows.

## Considered Options

1.  **`grandcentrix/zeroconf`**: A popular, actively maintained library for mDNS service discovery in Go.
2.  **`hashicorp/mdns`**: Another well-known mDNS library, used in projects like Consul.
3.  **Custom Implementation**: Building a custom mDNS client from scratch.

## Decision Outcome

Chosen option: **`grandcentrix/zeroconf`**, because it offers the best balance of features, active maintenance, and a straightforward API. It is widely used in the Go community and has proven to be reliable.

### Positive Consequences

- **Rapid Development**: Using a mature library accelerates development compared to a custom implementation.
- **Reliability**: The library is well-tested and used in many other projects.
- **Good Community Support**: Being a popular library, it's easier to find help and examples.

### Negative Consequences

- **External Dependency**: The project now depends on an external library, which introduces a maintenance and security consideration.
- **Library-Specific Quirks**: The implementation is tied to the specific API and behavior of the `zeroconf` library.

## Rationale

The `grandcentrix/zeroconf` library was chosen for its robust implementation and active maintenance. While `hashicorp/mdns` is also a solid choice, `zeroconf` has a slightly more intuitive API for the specific use case of browsing and resolving services. A custom implementation was rejected due to the complexity of the mDNS protocol and the availability of high-quality existing libraries.
