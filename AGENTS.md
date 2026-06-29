# kalypso - AI Agent Guide

## Design / Goal files

Refer to ./docs/design/

## Coding Standards & Best Practices

Refer to ./CODING_STANDARDS.md

## Hack folder

Ignore all files in ./hack.

## Knowledge Base

Refer to [KALYPSO_KNOWLEDGE.md](./KALYPSO_KNOWLEDGE.md) for compiled knowledge about KRO RGDs and Kalypso architecture.

## KRO CLI

Use `kro (generate|validate)` as part of the test harness to quickly validate a KRO RGD against test instances.

## Testing

- The testing is implemented using Kubernetes E2E Framework + KinD. It allows us to test the RGDs and CRDs againt a real running cluster.
- Tests also use KRO cli to validate the test data.
- Tests must not rely on external scripts (bash, Makefile, etc).
- The whole test suite must be run using `go test -v` and must not require pre-post steps to be performed manually.
- The KinD cluster should be created and torn down (conditionally via an env var - default to `yes`) as part of the test suite.
- On start, the test suite must tear down existing matching cluster to ensure tests run on a clean slate. Testing cluster must never be reused to ensure tests are properly run and won't be affected by previous runs.
- Resources created by the cluster should always be left behind to further troubleshooting can be performed. The resources will be deleted once the cluster itself is deleted.
- Once the e2e tests are run, perform checks on the live cluster every 20s to validate the state is progressing. If errors are found, perform the fixes and rerun the tests. The tests must be run with a timeout of 8 minutes.