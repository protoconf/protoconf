# E2E Tests

This directory contains end-to-end tests for Protoconf with multiple backend support.

## Overview

The e2e tests validate the complete workflow of Protoconf including:
- Configuration compilation
- Mutation operations
- Config subscription and updates
- Remote config loading
- Multiple KV backend support

## Supported Backends

The tests support the following KV backends:
- **Dummy**: In-memory backend (no external dependencies)
- **etcd**: Distributed key-value store
- **Consul**: Service mesh and KV store
- **Zookeeper**: Distributed coordination service

## Running Tests Locally

### Quick Start (Dummy Backend Only)

Run tests with the in-memory dummy backend (no infrastructure required):

```bash
go test -v ./test/...
```

### Running with All Backends

1. **Start the backend services:**
   ```bash
   docker-compose -f docker-compose.e2e.yml up -d
   ```

2. **Wait for services to be healthy:**
   ```bash
   # Check service status
   docker-compose -f docker-compose.e2e.yml ps

   # View logs if needed
   docker-compose -f docker-compose.e2e.yml logs
   ```

3. **Run the tests:**
   ```bash
   E2E_BACKENDS=all go test -v -run TestE2EMultiBackend ./test/...
   ```

4. **Clean up:**
   ```bash
   docker-compose -f docker-compose.e2e.yml down -v
   ```

### Running with Specific Backend

You can test against a specific backend by starting only that service:

**etcd:**
```bash
docker-compose -f docker-compose.e2e.yml up -d etcd
E2E_BACKENDS=all ETCD_ADDRESS=127.0.0.1:2379 go test -v -run TestE2EMultiBackend ./test/...
```

**Consul:**
```bash
docker-compose -f docker-compose.e2e.yml up -d consul
E2E_BACKENDS=all CONSUL_ADDRESS=127.0.0.1:8500 go test -v -run TestE2EMultiBackend ./test/...
```

**Zookeeper:**
```bash
docker-compose -f docker-compose.e2e.yml up -d zookeeper
E2E_BACKENDS=all ZOOKEEPER_ADDRESS=127.0.0.1:2181 go test -v -run TestE2EMultiBackend ./test/...
```

## Environment Variables

- `E2E_BACKENDS`: Set to "all" to enable tests with real backends (default: only dummy)
- `CI`: Set to "true" to enable all backends (used in CI/CD)
- `ETCD_ADDRESS`: etcd server address (default: 127.0.0.1:2379)
- `CONSUL_ADDRESS`: Consul server address (default: 127.0.0.1:8500)
- `ZOOKEEPER_ADDRESS`: Zookeeper server address (default: 127.0.0.1:2181)

## GitHub Actions

The e2e tests run automatically in GitHub Actions on PRs and pushes to main:

1. **e2e-dummy**: Fast tests with dummy backend (no infrastructure)
2. **e2e-backends**: Matrix job testing each backend individually
3. **e2e-all-together**: Tests with all backends running simultaneously

## Test Structure

The tests are organized in `e2e_multi_backend_test.go`:

- `BackendConfig`: Configuration for each backend type
- `createStore()`: Factory function for creating backend-specific stores
- `waitForBackend()`: Health check utility for real backends
- `runE2ETestWithBackend()`: Main test suite that runs against any backend

## Troubleshooting

### Services not starting

Check Docker logs:
```bash
docker-compose -f docker-compose.e2e.yml logs
```

### Port conflicts

If you have services running on the default ports (2379, 8500, 2181), you can modify the ports in `docker-compose.e2e.yml` and set the corresponding environment variables.

### Tests timing out

Some backends may take longer to start. Increase the timeout:
```bash
go test -v -run TestE2EMultiBackend ./test/... -timeout 15m
```

## CI/CD Integration

The workflow is defined in `.github/workflows/e2e.yml` and runs:
- On every pull request
- On pushes to main branch
- With all backends to ensure compatibility

## Adding New Backends

To add support for a new backend:

1. Add the backend type to `BackendType` constants
2. Update `createStore()` to handle the new backend
3. Add the backend to `getBackendConfigs()`
4. Update `docker-compose.e2e.yml` with the new service
5. Update this README
