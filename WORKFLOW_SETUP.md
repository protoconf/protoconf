# GitHub Actions Workflow - E2E Tests

## Status

✅ The e2e workflow is already active at `.github/workflows/e2e-workflow.yml`

## Implementation Details

The e2e workflow uses **GitHub Actions service containers** instead of docker-compose for:
- Reliable, fast testing with built-in health checks
- Native lifecycle management by GitHub Actions
- No external dependencies required in the workflow

## Workflow Overview

The workflow provides comprehensive testing across all supported backends:

1. **e2e-dummy**: Fast tests with in-memory dummy backend (no infrastructure required)
2. **e2e-etcd**: Tests with etcd backend using GitHub Actions service container
3. **e2e-consul**: Tests with consul backend using GitHub Actions service container
4. **e2e-zookeeper**: Tests with zookeeper backend using GitHub Actions service container
5. **e2e-all-together**: Comprehensive test with all three backends running simultaneously

Each backend job uses native GitHub Actions service containers with built-in health checks, providing reliable and fast test execution.

## How It Works

### Service Container Configuration

Each backend test job defines service containers that GitHub Actions manages:

```yaml
services:
  etcd:
    image: quay.io/coreos/etcd:v3.5.14
    ports:
      - 2379:2379
    options: >-
      --health-cmd "etcdctl endpoint health"
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
```

### Benefits Over Docker Compose

- **Automatic lifecycle**: Services start before tests and stop after
- **Built-in health checks**: Tests only run when services are ready
- **Better isolation**: Each job gets its own service instances
- **Faster execution**: No docker-compose startup overhead
- **Cleaner logs**: Service output is automatically captured

## Running Tests Locally

For local development, use the docker-compose setup:

```bash
# Start services
docker-compose -f docker-compose.e2e.yml up -d

# Run tests
E2E_BACKENDS=all go test -v -run TestE2EMultiBackend ./test/...

# Cleanup
docker-compose -f docker-compose.e2e.yml down -v
```

Or run without infrastructure (dummy backend only):

```bash
go test -v -run TestE2EMultiBackend ./test/...
```

## Workflow Triggers

The workflow runs automatically on:
- All pull requests targeting the main branch
- All pushes to the main branch

## Service Container Addresses

In CI, services are accessible at:
- **etcd**: `localhost:2379`
- **consul**: `localhost:8500`
- **zookeeper**: `localhost:2181`

The test code automatically detects when running in CI and uses these addresses.
