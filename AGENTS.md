# kalypso - AI Agent Guide

## Design / Goal files

Refer to ./docs/design/

## Coding Standards & Best Practices

Refer to ./CODING_STANDARDS.md

## Hack folder

Ignore all files in ./hack.

## Knowledge Base

Refer to [KALYPSO_KNOWLEDGE.md](file:///Users/miguel.santos/Projects/personal/kalypso/KALYPSO_KNOWLEDGE.md) for compiled knowledge about KRO RGDs and Kalypso architecture.

## KRO CLI

Use `kro (generate|validate)` as part of the test harness to quickly validate a KRO RGD against test instances.

## Testing

- The testing is implemented using Kubernetes E2E Framework + KinD. It allows us to test the RGDs and CRDs againt a real running cluster.
- Tests also use KRO cli to validate the test data.
- Tests must not rely on external scripts (bash, Makefile, etc).
- The whole test suite must be run using `go test -v` and must not require pre-post steps to be performed manually.
- The KinD cluster should be created and torn down (conditionally via an env var - default to `yes`) as part of the test suite.
