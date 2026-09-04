# Phase 9: Unit Test Coverage & Infrastructure - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-31
**Phase:** 09-unit-test-coverage-infrastructure
**Areas discussed:** Shared test helpers scope, Test depth per package, Coverage threshold, KV store test strategy
**Mode:** Auto (--auto flag)

---

## Shared Test Helpers Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Extract common patterns to testutil/ | gRPC server setup, proto helpers, temp dirs | [auto] |
| Keep helpers inline per test file | No shared package, duplicate as needed | |
| Full test framework with fixtures | Comprehensive test harness with setup/teardown | |

**User's choice:** [auto] Extract common patterns to testutil/ (recommended default)
**Notes:** Matches TEST-14 requirement. Keeps it minimal — only patterns that are actually duplicated.

---

## Test Depth Per Package

| Option | Description | Selected |
|--------|-------------|----------|
| Happy paths + key error paths | Cover primary functions, sentinel errors, flagged TODOs | [auto] |
| Happy paths only | Minimal coverage, just ensure basic functionality | |
| Exhaustive coverage | Every branch, every error path | |

**User's choice:** [auto] Happy paths + key error paths (recommended default)
**Notes:** Balanced approach. TEST-16 requires error path coverage but not exhaustive.

---

## Coverage Threshold

| Option | Description | Selected |
|--------|-------------|----------|
| No hard threshold, track and report | Measure via codecov, don't block CI | [auto] |
| 50% minimum threshold | Block CI below 50% | |
| 70% minimum threshold | Block CI below 70% | |

**User's choice:** [auto] No hard threshold initially, track and report (recommended default)
**Notes:** Adding 9 new test files will significantly increase coverage. Enforce threshold after baseline established.

---

## KV Store Test Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Interface-level with mock/in-memory | dummykv tested directly, others via mocks | [auto] |
| Real backend integration tests | Spin up Consul/etcd/ZK in test | |
| Skip KV store testing | Too complex for unit tests | |

**User's choice:** [auto] Interface-level testing with mock/in-memory backends (recommended default)
**Notes:** Matches CGO_ENABLED=0 constraint and CI (no external services). dummykv and filekv test directly. configmaps and otelkv use interface mocks.

---

## Claude's Discretion

- Exact test case selection per package
- Subtest vs flat test function choice
- Mock implementation details for configmaps/otelkv
- Whether to split large test files

## Deferred Ideas

- Placeholder test fixes — Phase 10 (TEST-07 through TEST-10)
- E2E integration tests — Phase 10 (TEST-11 through TEST-13)
- Coverage percentage thresholds — evaluate post Phase 9+10
