# Instance TLS Per-Field Source Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current global inline/file-path TLS source behavior with independent source choices for CA material and client certificate/key material.

**Architecture:** Keep `use_ssl` as the only persisted on/off switch and keep the existing path fields. Backend validation rejects inline/path conflicts only within the same material slot, resolves path fields into PEM independently, and validates the effective CA plus effective client cert/key pair. The React form removes the global TLS source selector and uses local-only source selectors for the CA group and client certificate/key group.

**Tech Stack:** Go, proto-generated Go structs, Connect API validation, React, TypeScript, react-i18next, Vitest, pnpm, golangci-lint.

---

## File Structure

- Modify `backend/api/v1/instance_service_tls_test.go`: update tests from global-source semantics to same-slot conflict and cross-slot mixing semantics.
- Modify `backend/api/v1/instance_service.go`: change TLS write validation and normalization helpers.
- Modify `backend/plugin/db/util/ssl_test.go`: add coverage that path resolution overwrites only matching slots and preserves inline material in other slots.
- Modify `backend/plugin/db/util/ssl.go`: keep per-field resolution behavior; adjust only if the new test exposes old global assumptions.
- Modify `frontend/src/react/components/instance/tls.ts`: replace `LocalTlsSource` with `LocalTlsCaSource` and `LocalTlsClientCertSource` helpers.
- Modify `frontend/src/react/components/instance/common.test.ts`: test source inference and per-group clearing helpers.
- Modify `frontend/src/react/components/instance/SslCertificateForm.tsx`: replace the global source selector with CA and client certificate/key source selectors.
- Modify `frontend/src/react/components/instance/SslCertificateForm.test.tsx`: test System trust rendering and per-group source controls.
- Modify `frontend/src/react/components/instance/DataSourceForm.tsx`: keep separate local CA and client source state and update only the corresponding fields.
- Modify `frontend/src/react/locales/*.json` and `frontend/src/locales/*.json`: update TLS source labels from one global source label to CA/client group labels.

---

### Task 1: Backend Same-Slot Validation

**Files:**
- Modify: `backend/api/v1/instance_service_tls_test.go`
- Modify: `backend/api/v1/instance_service.go`

- [ ] **Step 1: Replace the old global mixed-source failing tests**

In `backend/api/v1/instance_service_tls_test.go`, replace `TestValidateDataSourceTLSWriteRejectsMixedExplicitMaterial`, `TestNormalizeDataSourceTLSClearsInlineWhenPathWins`, `TestNormalizeDataSourceTLSClearsStaleInlineWhenPathExists`, `TestNormalizeDataSourceTLSClearsPathsWhenSwitchingToInline`, and `TestValidateDataSourceTLSWriteRejectsInactiveInlineWithExistingPath` with tests that encode the new behavior:

```go
func TestValidateDataSourceTLSWriteRejectsSameSlotMixedMaterial(t *testing.T) {
	tests := []struct {
		name string
		ds   *storepb.DataSource
		mask []string
		want string
	}{
		{
			name: "ca",
			ds:   &storepb.DataSource{UseSsl: true, SslCa: "inline-ca", SslCaPath: "/tmp/ca.pem"},
			mask: []string{"ssl_ca", "ssl_ca_path"},
			want: "cannot set both ssl_ca and ssl_ca_path",
		},
		{
			name: "cert",
			ds:   &storepb.DataSource{UseSsl: true, SslCert: "inline-cert", SslCertPath: "/tmp/cert.pem"},
			mask: []string{"ssl_cert", "ssl_cert_path"},
			want: "cannot set both ssl_cert and ssl_cert_path",
		},
		{
			name: "key",
			ds:   &storepb.DataSource{UseSsl: true, SslKey: "inline-key", SslKeyPath: "/tmp/key.pem"},
			mask: []string{"ssl_key", "ssl_key_path"},
			want: "cannot set both ssl_key and ssl_key_path",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDataSourceTLSWrite(tc.ds, tc.ds, tc.mask)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestValidateDataSourceTLSWriteAllowsCrossSlotMixedMaterial(t *testing.T) {
	err := validateDataSourceTLSWrite(
		&storepb.DataSource{UseSsl: true, SslCa: validCAPEM, SslCertPath: "/tmp/cert.pem", SslKeyPath: "/tmp/key.pem"},
		&storepb.DataSource{UseSsl: true, SslCa: validCAPEM, SslCertPath: "/tmp/cert.pem", SslKeyPath: "/tmp/key.pem"},
		[]string{"ssl_ca", "ssl_cert_path", "ssl_key_path"},
	)
	require.NoError(t, err)
}

func TestNormalizeDataSourceTLSClearsSameSlotConflictsOnly(t *testing.T) {
	ds := &storepb.DataSource{
		UseSsl:      true,
		SslCa:       "inline-ca",
		SslCaPath:   "/tmp/ca.pem",
		SslCert:     "inline-cert",
		SslKeyPath:  "/tmp/key.pem",
	}
	normalizeDataSourceTLS(ds, []string{"ssl_ca_path", "ssl_key_path"})
	require.Empty(t, ds.GetSslCa())
	require.Equal(t, "/tmp/ca.pem", ds.GetSslCaPath())
	require.Equal(t, "inline-cert", ds.GetSslCert())
	require.Equal(t, "/tmp/key.pem", ds.GetSslKeyPath())
}
```

Define `validCAPEM` in the test file with this valid self-signed certificate PEM:

```go
const validCAPEM = `-----BEGIN CERTIFICATE-----
MIIDOTCCAiGgAwIBAgIQSRJrEpBGFc7tNb1fb5pKFzANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEA6Gba5tHV1dAKouAaXO3/ebDUU4rvwCUg/CNaJ2PT5xLD4N1Vcb8r
bFSW2HXKq+MPfVdwIKR/1DczEoAGf/JWQTW7EgzlXrCd3rlajEX2D73faWJekD0U
aUgz5vtrTXZ90BQL7WvRICd7FlEZ6FPOcPlumiyNmzUqtwGhO+9ad1W5BqJaRI6P
YfouNkwR6Na4TzSj5BrqUfP0FwDizKSJ0XXmh8g8G9mtwxOSN3Ru1QFc61Xyeluk
POGKBV/q6RBNklTNe0gI8usUMlYyoC7ytppNMW7X2vodAelSu25jgx2anj9fDVZu
h7AXF5+4nJS4AAt0n1lNY7nGSsdZas8PbQIDAQABo4GIMIGFMA4GA1UdDwEB/wQE
AwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud
DgQWBBStsdjh3/JCXXYlQryOrL4Sh7BW5TAuBgNVHREEJzAlggtleGFtcGxlLmNv
bYcEfwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG9w0BAQsFAAOCAQEAxWGI
5NhpF3nwwy/4yB4i/CwwSpLrWUa70NyhvprUBC50PxiXav1TeDzwzLx/o5HyNwsv
cxv3HdkLW59i/0SlJSrNnWdfZ19oTcS+6PtLoVyISgtyN6DpkKpdG1cOkW3Cy2P2
+tK/tKHRP1Y/Ra0RiDpOAmqn0gCOFGz8+lqDIor/T7MTpibL3IxqWfPrvfVRHL3B
grw/ZQTTIVjjh4JBSW3WyWgNo/ikC1lrVxzl4iPUGptxT36Cr7Zk2Bsg0XqwbOvK
5d+NTDREkSnUbie4GeutujmX3Dsx88UiV6UY/4lHJa6I5leHUNOHahRbpbWeOfs/
WkBKOclmOV2xlTVuPw==
-----END CERTIFICATE-----`
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./backend/api/v1 -run 'TestValidateDataSourceTLSWrite|TestNormalizeDataSourceTLS|TestTLSMaskPaths|TestValidateDataSourceTLSConfig' -count=1
```

Expected: FAIL because current validation rejects any explicit inline+path write and current normalization clears all inline material when any path exists.

- [ ] **Step 3: Implement same-slot validation**

In `backend/api/v1/instance_service.go`, replace `hasExplicitInlineTLSMaterial`/`hasExplicitPathTLSMaterial` based rejection with slot-specific checks:

```go
func validateDataSourceTLSWrite(requested, merged *storepb.DataSource, mask []string) error {
	if !merged.GetUseSsl() {
		return nil
	}
	for _, conflict := range []struct {
		inlineField string
		pathField   string
		inlineValue string
		pathValue   string
	}{
		{"ssl_ca", "ssl_ca_path", merged.GetSslCa(), merged.GetSslCaPath()},
		{"ssl_cert", "ssl_cert_path", merged.GetSslCert(), merged.GetSslCertPath()},
		{"ssl_key", "ssl_key_path", merged.GetSslKey(), merged.GetSslKeyPath()},
	} {
		if conflict.inlineValue == "" || conflict.pathValue == "" {
			continue
		}
		if tlsMaskContains(mask, conflict.inlineField) || tlsMaskContains(mask, conflict.pathField) {
			return errors.Errorf("cannot set both %s and %s", conflict.inlineField, conflict.pathField)
		}
	}
	return validateDataSourceTLSConfig(merged)
}
```

Update `validateDataSourceTLSConfig` so path absoluteness is checked whenever path fields are provided, and inline PEM validation still runs when inline PEM is present. Do not return early just because any path field exists.

- [ ] **Step 4: Implement same-slot normalization**

Change `normalizeDataSourceTLS` to clear only conflicting inactive values for fields included in the update mask:

```go
func normalizeDataSourceTLS(ds *storepb.DataSource, mask []string) {
	if !ds.GetUseSsl() {
		clearInlineTLSMaterial(ds)
		clearPathTLSMaterial(ds)
		return
	}
	if tlsMaskContains(mask, "ssl_ca") {
		ds.SslCaPath = ""
	}
	if tlsMaskContains(mask, "ssl_ca_path") {
		ds.SslCa = ""
	}
	if tlsMaskContains(mask, "ssl_cert") {
		ds.SslCertPath = ""
	}
	if tlsMaskContains(mask, "ssl_cert_path") {
		ds.SslCert = ""
	}
	if tlsMaskContains(mask, "ssl_key") {
		ds.SslKeyPath = ""
	}
	if tlsMaskContains(mask, "ssl_key_path") {
		ds.SslKey = ""
	}
}
```

- [ ] **Step 5: Verify backend API tests pass**

Run:

```bash
go test ./backend/api/v1 -run 'TestValidateDataSourceTLSWrite|TestNormalizeDataSourceTLS|TestTLSMaskPaths|TestValidateDataSourceTLSConfig' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit backend validation changes**

Run:

```bash
gofmt -w backend/api/v1/instance_service.go backend/api/v1/instance_service_tls_test.go
git add backend/api/v1/instance_service.go backend/api/v1/instance_service_tls_test.go
git commit -m "fix(instance): allow per-field TLS material sources"
```

---

### Task 2: TLS Resolution Cross-Slot Preservation

**Files:**
- Modify: `backend/plugin/db/util/ssl_test.go`
- Modify: `backend/plugin/db/util/ssl.go`

- [ ] **Step 1: Add failing resolver test**

Add this test to `backend/plugin/db/util/ssl_test.go`:

```go
func TestResolveTLSMaterialPreservesInlineCrossSlotMaterial(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(keyPath, []byte("path-key"), 0o600))

	ds := &storepb.DataSource{
		UseSsl:     true,
		SslCa:      "inline-ca",
		SslCert:    "inline-cert",
		SslKeyPath: keyPath,
	}

	resolved, err := ResolveTLSMaterial(ds)
	require.NoError(t, err)
	require.Equal(t, "inline-ca", resolved.GetSslCa())
	require.Equal(t, "inline-cert", resolved.GetSslCert())
	require.Equal(t, "path-key", resolved.GetSslKey())
	require.Empty(t, resolved.GetSslKeyPath())
	require.Equal(t, keyPath, ds.GetSslKeyPath())
}
```

- [ ] **Step 2: Run resolver tests and verify the new test status**

Run:

```bash
go test ./backend/plugin/db/util -run 'TestResolveTLSMaterial' -count=1
```

Expected: PASS if current resolver already behaves per-field. If it fails, the failure should show a global-source assumption in `ResolveTLSMaterial`.

- [ ] **Step 3: Adjust resolver only if needed**

If the new test fails, change `ResolveTLSMaterial` so each path field overwrites only the matching inline field and clears only the matching path field. It should not clear `SslCa`, `SslCert`, or `SslKey` because another path field exists.

- [ ] **Step 4: Commit resolver test or fix**

Run:

```bash
gofmt -w backend/plugin/db/util/ssl.go backend/plugin/db/util/ssl_test.go
go test ./backend/plugin/db/util -run 'TestResolveTLSMaterial' -count=1
git add backend/plugin/db/util/ssl.go backend/plugin/db/util/ssl_test.go
git commit -m "test(instance): cover mixed TLS material resolution"
```

---

### Task 3: React TLS Source Helpers

**Files:**
- Modify: `frontend/src/react/components/instance/tls.ts`
- Modify: `frontend/src/react/components/instance/common.test.ts`

- [ ] **Step 1: Write failing helper tests**

Replace `frontend/src/react/components/instance/common.test.ts` with tests for per-group helpers:

```ts
import { describe, expect, test } from "vitest";
import {
  applyLocalTlsCaSource,
  applyLocalTlsClientCertSource,
  getLocalTlsCaSource,
  getLocalTlsClientCertSource,
  SSL_UPDATE_MASK_FIELDS,
} from "./tls";

describe("TLS update mask fields", () => {
  test("includes the SSL path fields alongside inline material", () => {
    expect(SSL_UPDATE_MASK_FIELDS).toEqual([
      "use_ssl",
      "ssl_ca",
      "ssl_cert",
      "ssl_key",
      "ssl_ca_path",
      "ssl_cert_path",
      "ssl_key_path",
    ]);
  });
});

describe("TLS local source helpers", () => {
  test("treats empty CA material with SSL enabled as system trust", () => {
    expect(getLocalTlsCaSource({ useSsl: true })).toBe("SYSTEM_TRUST");
  });

  test("clears only CA fields when selecting system trust", () => {
    const next = applyLocalTlsCaSource(
      {
        useSsl: true,
        sslCa: "inline-ca",
        sslCaPath: "/tmp/ca.pem",
        sslCert: "inline-cert",
        sslKeyPath: "/tmp/key.pem",
      } as never,
      "SYSTEM_TRUST"
    );
    expect(next.sslCa).toBe("");
    expect(next.sslCaPath).toBe("");
    expect(next.sslCert).toBe("inline-cert");
    expect(next.sslKeyPath).toBe("/tmp/key.pem");
  });

  test("clears only client cert fields when selecting none", () => {
    const next = applyLocalTlsClientCertSource(
      {
        useSsl: true,
        sslCaPath: "/tmp/ca.pem",
        sslCert: "inline-cert",
        sslKey: "inline-key",
        sslCertPath: "/tmp/cert.pem",
        sslKeyPath: "/tmp/key.pem",
      } as never,
      "NONE"
    );
    expect(next.sslCaPath).toBe("/tmp/ca.pem");
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
    expect(next.sslCertPath).toBe("");
    expect(next.sslKeyPath).toBe("");
  });

  test("infers client certificate source from path presence flags", () => {
    expect(getLocalTlsClientCertSource({ useSsl: true, hasSslCertPath: true } as never)).toBe("FILE_PATH");
  });
});
```

- [ ] **Step 2: Run helper tests and verify they fail**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/components/instance/common.test.ts
```

Expected: FAIL because the helper functions do not exist yet.

- [ ] **Step 3: Implement helper types and functions**

In `frontend/src/react/components/instance/tls.ts`, replace `LocalTlsSource` with:

```ts
export const LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST = "SYSTEM_TRUST" as const;
export const LOCAL_TLS_CA_SOURCE_INLINE_PEM = "INLINE_PEM" as const;
export const LOCAL_TLS_CA_SOURCE_FILE_PATH = "FILE_PATH" as const;

export const LOCAL_TLS_CLIENT_CERT_SOURCE_NONE = "NONE" as const;
export const LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM = "INLINE_PEM" as const;
export const LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH = "FILE_PATH" as const;

export type LocalTlsCaSource =
  | typeof LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST
  | typeof LOCAL_TLS_CA_SOURCE_INLINE_PEM
  | typeof LOCAL_TLS_CA_SOURCE_FILE_PATH;

export type LocalTlsClientCertSource =
  | typeof LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
  | typeof LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM
  | typeof LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH;
```

Implement `getLocalTlsCaSource`, `getLocalTlsClientCertSource`, `applyLocalTlsCaSource`, `applyLocalTlsClientCertSource`, and `disableLocalTls` so CA changes clear only CA fields and client source changes clear only cert/key fields.

- [ ] **Step 4: Verify helper tests pass**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/components/instance/common.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit helper changes**

Run:

```bash
pnpm --dir frontend fix
git add frontend/src/react/components/instance/tls.ts frontend/src/react/components/instance/common.test.ts
git commit -m "feat(frontend): split TLS material source helpers"
```

---

### Task 4: React Form Per-Group Source Selectors

**Files:**
- Modify: `frontend/src/react/components/instance/SslCertificateForm.tsx`
- Modify: `frontend/src/react/components/instance/SslCertificateForm.test.tsx`
- Modify: `frontend/src/react/components/instance/DataSourceForm.tsx`
- Modify: `frontend/src/react/locales/en-US.json`
- Modify: `frontend/src/react/locales/es-ES.json`
- Modify: `frontend/src/react/locales/ja-JP.json`
- Modify: `frontend/src/react/locales/vi-VN.json`
- Modify: `frontend/src/react/locales/zh-CN.json`
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/es-ES.json`
- Modify: `frontend/src/locales/ja-JP.json`
- Modify: `frontend/src/locales/vi-VN.json`
- Modify: `frontend/src/locales/zh-CN.json`

- [ ] **Step 1: Write failing form test**

Extend `frontend/src/react/components/instance/SslCertificateForm.test.tsx` with:

```tsx
it("renders explicit CA and client certificate source controls", () => {
  render(
    <SslCertificateForm
      useSsl={true}
      caSource="SYSTEM_TRUST"
      onCaSourceChange={() => {}}
      clientCertSource="FILE_PATH"
      onClientCertSourceChange={() => {}}
      showKeyAndCert={true}
    />
  );

  expect(screen.getByText("data-source.ssl.ca-source.self")).toBeTruthy();
  expect(screen.getByText("data-source.ssl.ca-source.system-trust")).toBeTruthy();
  expect(screen.getByText("data-source.ssl.client-cert-source.self")).toBeTruthy();
  expect(screen.getByText("data-source.ssl.client-cert-source.file-path")).toBeTruthy();
});
```

- [ ] **Step 2: Run form test and verify it fails**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/components/instance/SslCertificateForm.test.tsx
```

Expected: FAIL because `caSource` and `clientCertSource` props are not defined.

- [ ] **Step 3: Update form props and rendering**

In `SslCertificateForm.tsx`, replace `source/onSourceChange` with:

```ts
caSource?: LocalTlsCaSource;
onCaSourceChange?: (val: LocalTlsCaSource) => void;
clientCertSource?: LocalTlsClientCertSource;
onClientCertSourceChange?: (val: LocalTlsClientCertSource) => void;
```

Render `CA certificate source` radio options above CA material. For `SYSTEM_TRUST`, show no CA input and show the system trust hint. For `INLINE_PEM`, show the CA textarea. For `FILE_PATH`, show the CA path input.

Render `Client certificate source` radio options before client material when client cert/key fields are supported. For `NONE`, show no cert/key inputs. For `INLINE_PEM`, show the key/cert textareas. For `FILE_PATH`, show cert/key path inputs.

- [ ] **Step 4: Update DataSourceForm state wiring**

In `DataSourceForm.tsx`, replace `localTlsSource` state with:

```ts
const [localTlsCaSource, setLocalTlsCaSource] = useState(getLocalTlsCaSource(dataSource));
const [localTlsClientCertSource, setLocalTlsClientCertSource] = useState(getLocalTlsClientCertSource(dataSource));
```

On data source changes, sync both local states. Pass both sources to `SslCertificateForm`. On CA source change, call `applyLocalTlsCaSource`; on client source change, call `applyLocalTlsClientCertSource`. When SSL is disabled, call `disableLocalTls`.

- [ ] **Step 5: Update locale keys**

Add React and Vue locale keys:

```json
"data-source.ssl.ca-source.self": "CA Certificate Source",
"data-source.ssl.ca-source.system-trust": "System Trust",
"data-source.ssl.client-cert-source.self": "Client Certificate Source",
"data-source.ssl.client-cert-source.none": "None"
```

Reuse existing `data-source.ssl.source.inline-pem` and `data-source.ssl.source.file-path` labels unless the i18n check requires group-specific keys.

- [ ] **Step 6: Verify frontend tests pass**

Run:

```bash
pnpm --dir frontend fix
pnpm --dir frontend exec vitest run src/react/components/instance/common.test.ts src/react/components/instance/SslCertificateForm.test.tsx
pnpm --dir frontend check
pnpm --dir frontend type-check
```

Expected: all commands exit 0.

- [ ] **Step 7: Commit frontend form changes**

Run:

```bash
git add frontend/src/react/components/instance/SslCertificateForm.tsx frontend/src/react/components/instance/SslCertificateForm.test.tsx frontend/src/react/components/instance/DataSourceForm.tsx frontend/src/react/locales frontend/src/locales
git commit -m "feat(frontend): choose TLS sources per material group"
```

---

### Task 5: Final Verification

**Files:**
- Verify all files changed by Tasks 1 through 4.

- [ ] **Step 1: Run proto checks**

Run:

```bash
PATH="$(go env GOPATH)/bin:$PATH" buf format -w proto
PATH="$(go env GOPATH)/bin:$PATH" buf lint proto
(cd proto && PATH="$(go env GOPATH)/bin:$PATH" buf generate)
```

Expected: exit 0. If grpc-doc HTML files change only by whitespace, revert those generated-doc whitespace-only changes before committing.

- [ ] **Step 2: Run backend checks**

Run:

```bash
golangci-lint run --allow-parallel-runners
go test ./backend/api/v1 ./backend/plugin/db/util ./backend/component/dbfactory ./backend/component/ghost ./backend/plugin/db/elasticsearch -count=1
go test ./backend/plugin/db/mongodb -run '^TestBuildMongoshBaseArgsUsesTempDirForTLSFiles$' -count=1
```

Expected: exit 0. Do not run the full MongoDB package unless `mongosh` is installed.

- [ ] **Step 3: Run frontend checks**

Run:

```bash
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend exec vitest run src/react/components/instance/common.test.ts src/react/components/instance/SslCertificateForm.test.tsx
```

Expected: exit 0.

- [ ] **Step 4: Run server build**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: exit 0.

- [ ] **Step 5: Commit final verification fixes if needed**

If verification caused tracked changes, inspect them with:

```bash
git diff --stat
git diff --check
```

Then commit only intentional fixes:

```bash
git add <changed-files>
git commit -m "fix(instance): finalize TLS per-field source behavior"
```

Expected: `git status --short` is clean when done.
