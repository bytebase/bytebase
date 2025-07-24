# Lint Changed Files

Run linting and type checking on files that have been modified compared to the main branch.

## Steps:

1. **Identify changed files**
   - Get list of modified Go files: `git diff --name-only main...HEAD | grep '\.go$'`
   - Get list of modified frontend files: `git diff --name-only main...HEAD | grep '^frontend/' | grep -E '\.(ts|tsx|vue|js|jsx)$'`
   - Get list of modified proto files: `git diff --name-only main...HEAD | grep '\.proto$'`

2. **Run Go linting on changed files**
   ```bash
   # Get changed Go files
   changed_go_files=$(git diff --name-only main...HEAD | grep '\.go$')
   
   if [ -n "$changed_go_files" ]; then
     echo "Running golangci-lint on changed Go files..."
     golangci-lint run --allow-parallel-runners $changed_go_files
   else
     echo "No Go files changed"
   fi
   ```

3. **Run frontend linting and type checking**
   ```bash
   # Check if any frontend files changed
   if git diff --name-only main...HEAD | grep -q '^frontend/'; then
     echo "Running frontend lint..."
     pnpm --dir frontend lint
     
     echo "Running frontend type-check..."
     pnpm --dir frontend type-check
   else
     echo "No frontend files changed"
   fi
   ```

4. **Run protobuf formatting**
   ```bash
   # Check if any proto files changed
   if git diff --name-only main...HEAD | grep -q '\.proto$'; then
     echo "Running buf format on proto files..."
     buf format -w proto
   else
     echo "No proto files changed"
   fi
   ```

5. **Complete lint command**
   ```bash
   # Combined script to lint all changed files
   echo "Checking for changed files..."
   
   # Lint Go files
   changed_go_files=$(git diff --name-only main...HEAD | grep '\.go$' || true)
   if [ -n "$changed_go_files" ]; then
     echo "Linting Go files..."
     golangci-lint run --allow-parallel-runners
   fi
   
   # Lint and type-check frontend
   if git diff --name-only main...HEAD | grep -q '^frontend/' 2>/dev/null; then
     echo "Linting frontend files..."
     pnpm --dir frontend lint --fix
     pnpm --dir frontend type-check
   fi
   
   # Format proto files
   if git diff --name-only main...HEAD | grep -q '\.proto$' 2>/dev/null; then
     echo "Formatting proto files..."
     buf format -w proto
   fi
   
   echo "Lint check complete!"
   ```

## Notes:
- golangci-lint should be run without specifying individual files to avoid "function not defined" errors
- Frontend lint runs on the entire frontend directory when any frontend file changes
- Use `--fix` flag with frontend lint to auto-fix issues
- Run golangci-lint multiple times if needed until no issues remain (due to max-issues limit)
- buf format will automatically format all proto files in the proto directory when any proto file changes