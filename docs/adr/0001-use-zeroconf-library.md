# 1. Use Zeroconf Library for mDNS Discovery

* Status: accepted
* Date: 2025-11-12

## Context and Problem Statement

The application needs to discover Matter devices on the local network. Matter devices use mDNS (Multicast DNS) for discovery. We need a reliable and easy-to-use library for mDNS discovery in Go.

## Considered Options

* **grandcat/zeroconf**: A popular and well-maintained Go library for mDNS discovery.
* **brutella/dnssd**: Another Go library for DNS-SD, which is the basis for mDNS.
* **Custom implementation**: Writing our own mDNS client from scratch.

## Decision Outcome

Chosen option: "grandcat/zeroconf", because it is the most popular and well-maintained library for mDNS discovery in Go. It has a simple API and is easy to use. A custom implementation would be too complex and time-consuming.
