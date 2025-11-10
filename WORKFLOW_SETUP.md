# GitHub Actions Workflow Setup

## GitHub App Permission Issue

Due to GitHub App security restrictions, workflow files in `.github/workflows/` cannot be pushed automatically. The GitHub App requires the `workflows` permission to create or modify workflow files.

## Workflow File Location

The e2e workflow has been added to the repository as:
- **File**: `e2e-workflow.yml` (at repository root)
- **Target**: `.github/workflows/e2e.yml`

## Solutions

### Option 1: Move the File Manually (Recommended)

The workflow file is already in the repository, just needs to be moved:

```bash
# Move to correct location
git mv e2e-workflow.yml .github/workflows/e2e.yml
git commit -m "feat: enable e2e workflow in GitHub Actions"
git push
```

### Option 2: Add via GitHub Web Interface

1. Copy the content from `e2e-workflow.yml` in the repository
2. Go to GitHub → Your repo → `.github/workflows/`
3. Click "Add file" → "Create new file"
4. Name it `e2e.yml`
5. Paste the content and commit

### Option 3: Grant Workflow Permission to GitHub App

If you're using a GitHub App for this repository, grant it the `workflows` permission:

1. Go to GitHub Settings → Developer settings → GitHub Apps
2. Select your app
3. Under "Repository permissions" → "Workflows" → Set to "Read and write"
4. Save changes

## Implementation Details

The e2e workflow is included in this PR as `e2e-workflow.yml` to bypass GitHub App restrictions.

## Workflow Overview

The e2e.yml workflow provides three test jobs:

1. **e2e-dummy**: Fast tests with in-memory dummy backend (no infrastructure required)
2. **e2e-backends**: Matrix job testing each backend individually (etcd, consul, zookeeper)
3. **e2e-all-together**: Comprehensive test with all backends running simultaneously

## Testing Before Activation

Until the workflow is moved to `.github/workflows/`, you can still run e2e tests:

1. **Run tests locally** using the instructions in `test/E2E_README.md`
2. **Use docker-compose** with the provided `docker-compose.e2e.yml`
3. **Review workflow** by examining `e2e-workflow.yml` in the repository

## Next Steps

To activate the workflow (choose one):

1. **Simple move** (recommended):
   ```bash
   git mv e2e-workflow.yml .github/workflows/e2e.yml
   git commit -m "feat: enable e2e workflow in GitHub Actions"
   git push
   ```

2. **Add via GitHub web interface** (copy content from `e2e-workflow.yml`)

Once activated, the workflow will automatically run on:
- All pull requests targeting the main branch
- All pushes to the main branch
