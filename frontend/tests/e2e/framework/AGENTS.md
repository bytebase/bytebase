# E2E Framework — AI Agent Conventions

## File Structure

| File | Responsibility |
|------|----------------|
| `api-client.ts` | Bytebase v1 REST API wrapper with token refresh on 401 |
| `env.ts` | `TestEnv` interface, serialization to `.e2e-env.json` |
| `mode-start-new-bytebase.ts` | Start/stop disposable Bytebase with `--demo`, orphan cleanup |
| `global-setup.ts` | Start server, write partial env |
| `global-teardown.ts` | Stop server, cleanup |
| `setup-project.ts` | Playwright setup test: auth, instance/database discovery, env persistence |

## Adding a New Feature Test Suite

1. Create `tests/e2e/<feature>/` directory
2. Create page objects in the feature directory (not in `framework/`)
3. Create spec file importing `loadTestEnv` from `../framework/env`
4. In `beforeAll`: call `loadTestEnv()`, login, discover feature-specific data via API
5. Use API for state setup/teardown, browser for UI verification

## Conventions

- **Never hardcode** project, instance, or database names — discover via API
- **API for setup, browser for verification** — tests should be fast and deterministic
- **Page objects** belong in feature directories, not the framework
- The demo server uses `demo@example.com` as the admin account
- All tests share a single worker (`workers: 1`) for deterministic ordering

## Extending the API Client

Add methods to `BytebaseApiClient` in `api-client.ts`. Follow existing patterns:
- Use `this.request<T>()` with typed response
- Only add methods actually needed by tests
- Include `pageSize=100` on list endpoints
