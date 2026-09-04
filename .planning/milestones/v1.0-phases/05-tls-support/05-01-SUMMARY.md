---
phase: "05-tls-support"
plan: "01"
subsystem: "utils"
tags: ["tls", "security", "stdlib", "tdd"]
dependency_graph:
  requires: []
  provides: ["utils.TLSFiles", "utils.BuildTLSConfig"]
  affects: ["agent", "server", "mutate"]
tech_stack:
  added: []
  patterns: ["TDD red-green", "stdlib-only TLS", "X509KeyPair bytes-based parsing"]
key_files:
  created:
    - utils/tls.go
    - utils/tls_test.go
  modified: []
decisions:
  - "TLSFiles is a plain struct (not tied to proto types) for reuse across agent, server, and mutate CLI"
  - "Use tls.X509KeyPair with bytes (not tls.LoadX509KeyPair) to support both file and text PEM inputs"
  - "CAFile/CAText sets both ClientCAs pool and RequireAndVerifyClientCert for mutual TLS"
metrics:
  duration_seconds: 77
  completed_date: "2026-03-28"
  tasks_completed: 1
  files_created: 2
  files_modified: 0
---

# Phase 05 Plan 01: TLS Helper Summary

**One-liner:** Shared TLS helper using `tls.X509KeyPair` with `TLSFiles` struct supporting file paths and inline PEM text for cert, key, and CA.

## What Was Built

`utils/tls.go` exports two things:

1. `TLSFiles` — a plain struct with six string fields: `CertFile`, `CertText`, `KeyFile`, `KeyText`, `CAFile`, `CAText`. Package-agnostic to allow reuse by agent, mutation server, and mutate CLI.

2. `BuildTLSConfig(f TLSFiles) (*tls.Config, error)` — constructs a `*tls.Config` from the struct:
   - Returns `(nil, nil)` when neither cert nor key is provided
   - Returns error when only cert or only key is set ("cert and key must both be set")
   - Reads from file (via `os.ReadFile`) or uses inline text directly
   - Calls `tls.X509KeyPair(certPEM, keyPEM)` for parsing
   - When CAFile/CAText is set: creates `x509.NewCertPool()`, appends certs, sets `ClientCAs` and `ClientAuth = tls.RequireAndVerifyClientCert`

`utils/tls_test.go` provides:
- `generateSelfSignedCert(t)` helper — ECDSA P-256, valid 1 hour, IP SAN 127.0.0.1
- `TestBuildTLSConfig` with 11 table-driven subtests covering all input combinations

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| RED  | ce983d5 | test(05-01): add failing tests for BuildTLSConfig |
| GREEN | c9def46 | feat(05-01): implement BuildTLSConfig TLS helper |

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all paths are fully implemented.

## Self-Check: PASSED

- `utils/tls.go` exists: FOUND
- `utils/tls_test.go` exists: FOUND
- commit ce983d5 exists: FOUND
- commit c9def46 exists: FOUND
- All 11 TestBuildTLSConfig subtests pass
- No InsecureSkipVerify
- No os.Exit
