# GitHub Actions Workflow Setup

## GitHub App Permission Issue

Due to GitHub App security restrictions, the workflow file `.github/workflows/e2e.yml` cannot be pushed automatically. The GitHub App requires the `workflows` permission to create or modify workflow files.

## Solutions

### Option 1: Manually Add the Workflow File

Copy the workflow file from your local repository to GitHub:

1. The workflow file is already created locally at `.github/workflows/e2e.yml`
2. You can either:
   - Commit and push it using a personal access token or SSH key
   - Manually create the file on GitHub through the web interface

### Option 2: Grant Workflow Permission to GitHub App

If you're using a GitHub App for this repository, grant it the `workflows` permission:

1. Go to GitHub Settings → Developer settings → GitHub Apps
2. Select your app
3. Under "Repository permissions" → "Workflows" → Set to "Read and write"
4. Save changes

## Workflow File Location

The complete workflow file is available at:
- **Local path**: `.github/workflows/e2e.yml`
- **Size**: ~154 lines
- **Format**: YAML

## Workflow Overview

The e2e.yml workflow provides three test jobs:

1. **e2e-dummy**: Fast tests with in-memory dummy backend (no infrastructure required)
2. **e2e-backends**: Matrix job testing each backend individually (etcd, consul, zookeeper)
3. **e2e-all-together**: Comprehensive test with all backends running simultaneously

## Temporary Workaround

Until the workflow file is added to the repository, you can:

1. **Run tests locally** using the instructions in `test/E2E_README.md`
2. **Manually trigger tests** using docker-compose and go test commands
3. **Copy the workflow content** from `.github/workflows/e2e.yml` and add it through GitHub's web interface

## Next Steps

After resolving the permission issue:

```bash
# Verify the workflow file exists locally
ls -la .github/workflows/e2e.yml

# If using personal credentials, push directly:
git add .github/workflows/e2e.yml
git commit -m "feat: add GitHub Actions workflow for e2e tests"
git push
```

The workflow will then automatically run on:
- All pull requests targeting the main branch
- All pushes to the main branch
