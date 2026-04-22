# Instance TLS UI Posture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the instance data source TLS UI with a posture-first `Connection security` design while preserving the existing backend/API/storage model.

**Architecture:** Add local TLS posture helpers in `frontend/src/react/components/instance/tls.ts`, add a small segmented-control primitive for the posture/source controls, and update `SslCertificateForm` plus `DataSourceForm` to map the UI posture onto existing `useSsl`, CA, and client certificate/key fields. Vault TLS keeps the existing legacy path because it does not pass posture props.

**Tech Stack:** React, Base UI-compatible local UI primitives, Tailwind CSS v4 classes, Vitest, `react-i18next`, Pinia state bridged with `useVueState`.

---

## File Structure

- Modify `frontend/src/react/components/instance/tls.ts`: add posture constants/types and helper functions for posture inference, posture application, and client identity support.
- Modify `frontend/src/react/components/instance/common.test.ts`: add tests for posture inference and clearing behavior.
- Create `frontend/src/react/components/ui/segmented-control.tsx`: reusable segmented control with selected, disabled, and tooltip-capable options.
- Modify `frontend/src/react/components/instance/SslCertificateForm.tsx`: replace instance-mode switch/radio source UI with posture and segmented source controls; keep legacy rendering for Vault callers.
- Modify `frontend/src/react/components/instance/SslCertificateForm.test.tsx`: update existing tests and add component tests for posture, SaaS file path disabling, verification explanation, and unsupported mTLS.
- Modify `frontend/src/react/components/instance/DataSourceForm.tsx`: infer and sync local posture state, pass SaaS mode, and map posture changes to existing data source fields.
- Modify locale files under `frontend/src/react/locales/`: add the new display strings to every React locale file.

---

### Task 1: Add TLS Posture Helpers

**Files:**
- Modify: `frontend/src/react/components/instance/tls.ts`
- Test: `frontend/src/react/components/instance/common.test.ts`

- [ ] **Step 1: Write failing posture helper tests**

Add these imports in `frontend/src/react/components/instance/common.test.ts`:

```ts
import { Engine } from "@/types/proto-es/v1/common_pb";
```

Extend the existing TLS imports:

```ts
import {
  applyLocalTlsCaSource,
  applyLocalTlsClientCertSource,
  applyLocalTlsPosture,
  getLocalTlsCaSource,
  getLocalTlsClientCertSource,
  getLocalTlsPosture,
  isLocalTlsClientIdentitySupported,
  SSL_UPDATE_MASK_FIELDS,
} from "./tls";
```

Append these tests to `frontend/src/react/components/instance/common.test.ts`:

```ts
describe("TLS posture helpers", () => {
  test("infers disabled posture when SSL is off", () => {
    expect(getLocalTlsPosture({ useSsl: false })).toBe("DISABLED");
  });

  test("infers TLS posture when SSL is on without client identity", () => {
    expect(
      getLocalTlsPosture({
        useSsl: true,
        sslCaPathSet: true,
      } as never)
    ).toBe("TLS");
  });

  test("infers mutual TLS posture from inline client material", () => {
    expect(
      getLocalTlsPosture({
        useSsl: true,
        sslCertSet: true,
        sslKeySet: true,
      } as never)
    ).toBe("MUTUAL_TLS");
  });

  test("infers mutual TLS posture from file path client material", () => {
    expect(
      getLocalTlsPosture({
        useSsl: true,
        sslCertPathSet: true,
        sslKeyPathSet: true,
      } as never)
    ).toBe("MUTUAL_TLS");
  });

  test("switching posture to TLS clears only client identity fields", () => {
    const next = applyLocalTlsPosture(
      {
        useSsl: true,
        sslCaPath: "/tmp/ca.pem",
        sslCaPathSet: true,
        sslCert: "inline-cert",
        sslKey: "inline-key",
        sslCertPath: "/tmp/cert.pem",
        sslKeyPath: "/tmp/key.pem",
        sslCertSet: true,
        sslKeySet: true,
        sslCertPathSet: true,
        sslKeyPathSet: true,
      } as never,
      "TLS"
    );

    expect(next.useSsl).toBe(true);
    expect(next.sslCaPath).toBe("/tmp/ca.pem");
    expect(next.sslCaPathSet).toBe(true);
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
    expect(next.sslCertPath).toBe("");
    expect(next.sslKeyPath).toBe("");
    expect(next.sslCertSet).toBe(false);
    expect(next.sslKeySet).toBe(false);
    expect(next.sslCertPathSet).toBe(false);
    expect(next.sslKeyPathSet).toBe(false);
  });

  test("switching posture to disabled clears all TLS material", () => {
    const next = applyLocalTlsPosture(
      {
        useSsl: true,
        sslCa: "inline-ca",
        sslCaPath: "/tmp/ca.pem",
        sslCert: "inline-cert",
        sslKey: "inline-key",
      } as never,
      "DISABLED"
    );

    expect(next.useSsl).toBe(false);
    expect(next.sslCa).toBe("");
    expect(next.sslCaPath).toBe("");
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
  });

  test("MSSQL does not support client identity in this form", () => {
    expect(isLocalTlsClientIdentitySupported(Engine.MSSQL)).toBe(false);
    expect(isLocalTlsClientIdentitySupported(Engine.POSTGRES)).toBe(true);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
pnpm --dir frontend test -- SslCertificateForm common
```

Expected: FAIL because `applyLocalTlsPosture`, `getLocalTlsPosture`, and `isLocalTlsClientIdentitySupported` are not exported yet.

- [ ] **Step 3: Implement posture helpers**

In `frontend/src/react/components/instance/tls.ts`, add the engine import near the top:

```ts
import { Engine } from "@/types/proto-es/v1/common_pb";
```

Add posture constants and type after the existing local TLS source constants:

```ts
export const LOCAL_TLS_POSTURE_DISABLED = "DISABLED" as const;
export const LOCAL_TLS_POSTURE_TLS = "TLS" as const;
export const LOCAL_TLS_POSTURE_MUTUAL_TLS = "MUTUAL_TLS" as const;

export type LocalTlsPosture =
  | typeof LOCAL_TLS_POSTURE_DISABLED
  | typeof LOCAL_TLS_POSTURE_TLS
  | typeof LOCAL_TLS_POSTURE_MUTUAL_TLS;
```

Add these helpers after `getLocalTlsClientCertSource`:

```ts
export const getLocalTlsPosture = (
  ds: LocalTlsDataSource | undefined
): LocalTlsPosture => {
  if (!ds?.useSsl) {
    return LOCAL_TLS_POSTURE_DISABLED;
  }
  return getLocalTlsClientCertSource(ds) === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
    ? LOCAL_TLS_POSTURE_TLS
    : LOCAL_TLS_POSTURE_MUTUAL_TLS;
};

export const isLocalTlsClientIdentitySupported = (engine: Engine): boolean => {
  return engine !== Engine.MSSQL;
};
```

Add this helper after `applyLocalTlsClientCertSource`:

```ts
export const applyLocalTlsPosture = (
  ds: DataSource,
  posture: LocalTlsPosture
): DataSource => {
  const next = cloneDeep(ds);
  switch (posture) {
    case LOCAL_TLS_POSTURE_DISABLED:
      return disableLocalTls(next);
    case LOCAL_TLS_POSTURE_TLS:
      next.useSsl = true;
      clearLocalTlsClientCertFields(next);
      return next;
    case LOCAL_TLS_POSTURE_MUTUAL_TLS:
      next.useSsl = true;
      return next;
  }
};
```

- [ ] **Step 4: Run helper tests**

Run:

```bash
pnpm --dir frontend test -- common
```

Expected: PASS for `common.test.ts`.

- [ ] **Step 5: Commit helper changes**

Run:

```bash
git add frontend/src/react/components/instance/tls.ts frontend/src/react/components/instance/common.test.ts
git commit -m "feat: add local TLS posture helpers"
```

---

### Task 2: Add a Segmented Control Primitive

**Files:**
- Create: `frontend/src/react/components/ui/segmented-control.tsx`
- Test indirectly in: `frontend/src/react/components/instance/SslCertificateForm.test.tsx`

- [ ] **Step 1: Create the segmented control**

Create `frontend/src/react/components/ui/segmented-control.tsx`:

```tsx
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";

export interface SegmentedControlOption<T extends string> {
  value: T;
  label: React.ReactNode;
  disabled?: boolean;
  tooltip?: React.ReactNode;
}

interface SegmentedControlProps<T extends string> {
  value: T;
  options: SegmentedControlOption<T>[];
  onValueChange: (value: T) => void;
  ariaLabel: string;
  disabled?: boolean;
  className?: string;
}

export function SegmentedControl<T extends string>({
  value,
  options,
  onValueChange,
  ariaLabel,
  disabled = false,
  className,
}: SegmentedControlProps<T>) {
  return (
    <div
      role="radiogroup"
      aria-label={ariaLabel}
      className={cn(
        "inline-flex max-w-full flex-wrap rounded-xs border border-control-border bg-background",
        className
      )}
    >
      {options.map((option, index) => {
        const selected = option.value === value;
        const optionDisabled = disabled || option.disabled;
        const button = (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={selected}
            aria-disabled={optionDisabled || undefined}
            data-state={selected ? "checked" : "unchecked"}
            data-disabled={optionDisabled || undefined}
            className={cn(
              "min-h-8 px-3 text-sm transition-colors focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
              index > 0 && "border-l border-control-border",
              selected
                ? "bg-accent text-accent-text"
                : "bg-background text-control hover:bg-control-bg",
              optionDisabled && "cursor-not-allowed opacity-50 hover:bg-background"
            )}
            onClick={() => {
              if (!optionDisabled) {
                onValueChange(option.value);
              }
            }}
          >
            {option.label}
          </button>
        );

        if (!option.tooltip) {
          return button;
        }
        return (
          <Tooltip key={option.value} content={option.tooltip}>
            {button}
          </Tooltip>
        );
      })}
    </div>
  );
}
```

- [ ] **Step 2: Run frontend type check for the new component**

Run:

```bash
pnpm --dir frontend type-check
```

Expected: PASS.

- [ ] **Step 3: Commit segmented control**

Run:

```bash
git add frontend/src/react/components/ui/segmented-control.tsx
git commit -m "feat: add React segmented control"
```

---

### Task 3: Update Locale Strings

**Files:**
- Modify: `frontend/src/react/locales/en-US.json`
- Modify: `frontend/src/react/locales/zh-CN.json`
- Modify: `frontend/src/react/locales/ja-JP.json`
- Modify: `frontend/src/react/locales/es-ES.json`
- Modify: `frontend/src/react/locales/vi-VN.json`

- [ ] **Step 1: Update the English locale**

In `frontend/src/react/locales/en-US.json`, update the `data-source.ssl` object:

```json
{
  "ca-empty-uses-system-trust": "Uses the system trust store to verify the server certificate.",
  "ca-source": {
    "file-path": "File path",
    "file-path-unavailable-saas": "File paths are unavailable in Bytebase Cloud.",
    "inline-pem": "Paste PEM",
    "self": "CA certificate source",
    "system-trust": "System trust"
  },
  "client-cert": "Certificate",
  "client-cert-path": "Certificate path",
  "client-cert-source": {
    "file-path": "File path",
    "file-path-unavailable-saas": "File paths are unavailable in Bytebase Cloud.",
    "inline-pem": "Paste PEM",
    "none": "None",
    "self": "Client identity source"
  },
  "client-identity": "Client identity",
  "client-key": "Private key",
  "client-key-path": "Private key path",
  "connection-security": "Connection security",
  "mutual-tls-unavailable-engine": "Mutual TLS is not available for this engine.",
  "posture": {
    "disabled": "Disabled",
    "mutual-tls": "Mutual TLS",
    "self": "Security posture",
    "tls": "TLS"
  },
  "server-identity": "Server identity",
  "verification-disabled-description": "Verification is disabled. The connection is encrypted, but the server identity is not verified; CA settings are ignored."
}
```

Preserve existing keys not shown in this snippet, such as placeholders and `configured`.

- [ ] **Step 2: Update non-English React locale files**

Apply the same key structure to:

```bash
frontend/src/react/locales/zh-CN.json
frontend/src/react/locales/ja-JP.json
frontend/src/react/locales/es-ES.json
frontend/src/react/locales/vi-VN.json
```

Use the English strings as fallback values if a proper translation is not available. Do not add empty objects.

- [ ] **Step 3: Validate locale JSON**

Run:

```bash
pnpm --dir frontend check
```

Expected: PASS.

- [ ] **Step 4: Commit locale changes**

Run:

```bash
git add frontend/src/react/locales/en-US.json frontend/src/react/locales/zh-CN.json frontend/src/react/locales/ja-JP.json frontend/src/react/locales/es-ES.json frontend/src/react/locales/vi-VN.json
git commit -m "chore: add TLS posture locale strings"
```

---

### Task 4: Redesign `SslCertificateForm`

**Files:**
- Modify: `frontend/src/react/components/instance/SslCertificateForm.tsx`
- Test: `frontend/src/react/components/instance/SslCertificateForm.test.tsx`

- [ ] **Step 1: Write failing component tests**

Replace the current explicit-source test in `frontend/src/react/components/instance/SslCertificateForm.test.tsx` with tests that describe the new UI. Keep the existing `marks write-only TLS material as configured` test and update expected labels as needed.

Add these tests:

```tsx
import type { ReactNode } from "react";

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({
    content,
    children,
  }: {
    content: ReactNode;
    children: ReactNode;
  }) => (
    <span data-tooltip={typeof content === "string" ? content : undefined}>
      {children}
      {content}
    </span>
  ),
}));

test("renders posture-first connection security controls", () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  act(() => {
    root.render(
      <SslCertificateForm
        posture="TLS"
        onPostureChange={() => {}}
        caSource="SYSTEM_TRUST"
        onCaSourceChange={() => {}}
        clientCertSource="NONE"
        onClientCertSourceChange={() => {}}
        useSsl={true}
        verify={true}
        onVerifyChange={() => {}}
        engineType={Engine.POSTGRES}
      />
    );
  });

  expect(container.textContent).toContain("data-source.ssl.connection-security");
  expect(container.textContent).toContain("data-source.ssl.posture.disabled");
  expect(container.textContent).toContain("data-source.ssl.posture.tls");
  expect(container.textContent).toContain("data-source.ssl.posture.mutual-tls");
  expect(container.textContent).toContain("data-source.ssl.server-identity");
  expect(container.textContent).toContain(
    "data-source.ssl.ca-empty-uses-system-trust"
  );
  expect(container.textContent).not.toContain("data-source.ssl.client-identity");

  act(() => {
    root.unmount();
  });
});

test("renders client identity for mutual TLS without a None source option", () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  act(() => {
    root.render(
      <SslCertificateForm
        posture="MUTUAL_TLS"
        onPostureChange={() => {}}
        caSource="SYSTEM_TRUST"
        onCaSourceChange={() => {}}
        clientCertSource="INLINE_PEM"
        onClientCertSourceChange={() => {}}
        useSsl={true}
        verify={true}
        onVerifyChange={() => {}}
        showKeyAndCert
        engineType={Engine.POSTGRES}
      />
    );
  });

  expect(container.textContent).toContain("data-source.ssl.client-identity");
  expect(container.textContent).toContain(
    "data-source.ssl.client-cert-source.inline-pem"
  );
  expect(container.textContent).toContain(
    "data-source.ssl.client-cert-source.file-path"
  );
  expect(container.textContent).not.toContain(
    "data-source.ssl.client-cert-source.none"
  );

  act(() => {
    root.unmount();
  });
});

test("keeps CA controls visible when verification is disabled", () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  act(() => {
    root.render(
      <SslCertificateForm
        posture="TLS"
        onPostureChange={() => {}}
        caSource="INLINE_PEM"
        onCaSourceChange={() => {}}
        clientCertSource="NONE"
        onClientCertSourceChange={() => {}}
        useSsl={true}
        verify={false}
        onVerifyChange={() => {}}
        engineType={Engine.POSTGRES}
      />
    );
  });

  expect(container.textContent).toContain(
    "data-source.ssl.verification-disabled-description"
  );
  expect(container.textContent).toContain("data-source.ssl.ca-source.self");

  act(() => {
    root.unmount();
  });
});

test("disables file path source options in SaaS mode", () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  act(() => {
    root.render(
      <SslCertificateForm
        posture="MUTUAL_TLS"
        onPostureChange={() => {}}
        caSource="FILE_PATH"
        onCaSourceChange={() => {}}
        clientCertSource="FILE_PATH"
        onClientCertSourceChange={() => {}}
        useSsl={true}
        verify={true}
        onVerifyChange={() => {}}
        isSaaSMode
        showKeyAndCert
        engineType={Engine.POSTGRES}
      />
    );
  });

  expect(
    container.querySelector('[aria-label="data-source.ssl.ca-source.self"] [aria-disabled="true"][aria-checked="true"]')
  ).not.toBeNull();
  expect(container.textContent).toContain(
    "data-source.ssl.ca-source.file-path-unavailable-saas"
  );

  act(() => {
    root.unmount();
  });
});

test("shows disabled mutual TLS for unsupported engines", () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  act(() => {
    root.render(
      <SslCertificateForm
        posture="TLS"
        onPostureChange={() => {}}
        caSource="SYSTEM_TRUST"
        onCaSourceChange={() => {}}
        clientCertSource="NONE"
        onClientCertSourceChange={() => {}}
        useSsl={true}
        verify={true}
        onVerifyChange={() => {}}
        engineType={Engine.MSSQL}
      />
    );
  });

  expect(container.textContent).toContain(
    "data-source.ssl.mutual-tls-unavailable-engine"
  );
  expect(
    container.querySelector('[aria-disabled="true"]')
  ).not.toBeNull();

  act(() => {
    root.unmount();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
pnpm --dir frontend test -- SslCertificateForm
```

Expected: FAIL because `SslCertificateForm` does not yet accept posture or SaaS props and still renders radio source selectors.

- [ ] **Step 3: Update imports and props**

In `frontend/src/react/components/instance/SslCertificateForm.tsx`, remove the radio group import:

```ts
-import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
```

Add:

```ts
import { SegmentedControl } from "@/react/components/ui/segmented-control";
```

Extend the TLS imports:

```ts
import {
  getLocalTlsCaSource,
  getLocalTlsClientCertSource,
  isLocalTlsClientIdentitySupported,
  LOCAL_TLS_CA_SOURCE_FILE_PATH,
  LOCAL_TLS_CA_SOURCE_INLINE_PEM,
  LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST,
  LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH,
  LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM,
  LOCAL_TLS_CLIENT_CERT_SOURCE_NONE,
  LOCAL_TLS_POSTURE_DISABLED,
  LOCAL_TLS_POSTURE_MUTUAL_TLS,
  LOCAL_TLS_POSTURE_TLS,
  type LocalTlsCaSource,
  type LocalTlsClientCertSource,
  type LocalTlsPosture,
} from "./tls";
```

Add props to `SslCertificateFormProps`:

```ts
  posture?: LocalTlsPosture;
  onPostureChange?: (val: LocalTlsPosture) => void;
  isSaaSMode?: boolean;
```

Destructure them in `SslCertificateForm`:

```ts
  posture,
  onPostureChange,
  isSaaSMode = false,
```

- [ ] **Step 4: Replace source selector helpers with segmented controls**

Replace `CaSourceSelector` with:

```tsx
function CaSourceSelector({
  value,
  onChange,
  disabled = false,
  isSaaSMode = false,
}: {
  value: LocalTlsCaSource;
  onChange: (value: LocalTlsCaSource) => void;
  disabled?: boolean;
  isSaaSMode?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <SegmentedControl
      value={value}
      onValueChange={onChange}
      ariaLabel={t("data-source.ssl.ca-source.self")}
      disabled={disabled}
      options={[
        {
          value: LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST,
          label: t("data-source.ssl.ca-source.system-trust"),
        },
        {
          value: LOCAL_TLS_CA_SOURCE_INLINE_PEM,
          label: t("data-source.ssl.ca-source.inline-pem"),
        },
        {
          value: LOCAL_TLS_CA_SOURCE_FILE_PATH,
          label: t("data-source.ssl.ca-source.file-path"),
          disabled: isSaaSMode,
          tooltip: isSaaSMode
            ? t("data-source.ssl.ca-source.file-path-unavailable-saas")
            : undefined,
        },
      ]}
    />
  );
}
```

Replace `ClientCertSourceSelector` with:

```tsx
function ClientCertSourceSelector({
  value,
  onChange,
  disabled = false,
  isSaaSMode = false,
}: {
  value: LocalTlsClientCertSource;
  onChange: (value: LocalTlsClientCertSource) => void;
  disabled?: boolean;
  isSaaSMode?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <SegmentedControl
      value={value}
      onValueChange={onChange}
      ariaLabel={t("data-source.ssl.client-cert-source.self")}
      disabled={disabled}
      options={[
        {
          value: LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM,
          label: t("data-source.ssl.client-cert-source.inline-pem"),
        },
        {
          value: LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH,
          label: t("data-source.ssl.client-cert-source.file-path"),
          disabled: isSaaSMode,
          tooltip: isSaaSMode
            ? t("data-source.ssl.client-cert-source.file-path-unavailable-saas")
            : undefined,
        },
      ]}
    />
  );
}
```

- [ ] **Step 5: Add posture rendering**

Inside `SslCertificateForm`, compute posture state:

```ts
  const showPostureUi = posture !== undefined && !!onPostureChange;
  const resolvedPosture =
    posture ??
    (resolvedUseSsl
      ? resolvedClientCertSource === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
        ? LOCAL_TLS_POSTURE_TLS
        : LOCAL_TLS_POSTURE_MUTUAL_TLS
      : LOCAL_TLS_POSTURE_DISABLED);
  const supportsClientIdentity =
    showKeyAndCertFields && isLocalTlsClientIdentitySupported(engineType);
  const savedClientIdentity =
    resolvedClientCertSource !== LOCAL_TLS_CLIENT_CERT_SOURCE_NONE;
  const canSelectMutualTls = supportsClientIdentity || savedClientIdentity;
```

Add this JSX helper before the `return`:

```tsx
  const renderPostureControl = () => {
    if (!showPostureUi) {
      return null;
    }
    return (
      <div className="flex flex-col gap-y-1">
        <label className="textlabel block">
          {t("data-source.ssl.posture.self")}
        </label>
        <SegmentedControl
          value={resolvedPosture}
          onValueChange={onPostureChange}
          ariaLabel={t("data-source.ssl.posture.self")}
          disabled={disabled}
          options={[
            {
              value: LOCAL_TLS_POSTURE_DISABLED,
              label: t("data-source.ssl.posture.disabled"),
            },
            {
              value: LOCAL_TLS_POSTURE_TLS,
              label: t("data-source.ssl.posture.tls"),
            },
            {
              value: LOCAL_TLS_POSTURE_MUTUAL_TLS,
              label: t("data-source.ssl.posture.mutual-tls"),
              disabled: !canSelectMutualTls,
              tooltip: !canSelectMutualTls
                ? t("data-source.ssl.mutual-tls-unavailable-engine")
                : undefined,
            },
          ]}
        />
      </div>
    );
  };
```

In the main return, render the new section title and posture control when `showPostureUi` is true:

```tsx
    <div className="mt-2 flex flex-col gap-y-3">
      {showPostureUi && (
        <label className="textlabel block">
          {t("data-source.ssl.connection-security")}
        </label>
      )}
      {renderPostureControl()}
      {!showPostureUi && showUseSslSwitch && (
        // existing switch block
      )}
```

Keep the legacy `showUseSslSwitch` branch only when `!showPostureUi`.

- [ ] **Step 6: Render server and client identity groups**

In the posture branch, replace the flat per-group source UI with grouped sections:

```tsx
              {showPostureUi ? (
                <>
                  {resolvedPosture !== LOCAL_TLS_POSTURE_DISABLED && (
                    <fieldset className="rounded-xs border border-control-border p-3">
                      <legend className="px-1 text-xs font-medium text-control-light">
                        {t("data-source.ssl.server-identity")}
                      </legend>
                      <div className="flex flex-col gap-y-3">
                        {showVerify && (
                          <div className="flex flex-row items-center gap-x-1">
                            <Switch
                              checked={verify}
                              onCheckedChange={(val) => onVerifyChange?.(val)}
                              disabled={disabled}
                            />
                            <label className="textlabel block">
                              {resolvedVerifyLabel}
                            </label>
                            {showTooltip && (
                              <Tooltip
                                content={t(
                                  "data-source.ssl.verify-certificate-tooltip"
                                )}
                                side="right"
                              >
                                <Info className="size-4 text-warning" />
                              </Tooltip>
                            )}
                          </div>
                        )}
                        {!verify && (
                          <p className="text-xs text-warning">
                            {t(
                              "data-source.ssl.verification-disabled-description"
                            )}
                          </p>
                        )}
                        {showCaSourceUi && (
                          <div className="flex flex-col gap-y-1">
                            <label className="textlabel block">
                              {t("data-source.ssl.ca-source.self")}
                            </label>
                            <CaSourceSelector
                              value={resolvedCaSource}
                              onChange={onCaSourceChange}
                              disabled={disabled}
                              isSaaSMode={isSaaSMode}
                            />
                          </div>
                        )}
                        {renderCaMaterial()}
                      </div>
                    </fieldset>
                  )}

                  {resolvedPosture === LOCAL_TLS_POSTURE_MUTUAL_TLS && (
                    <fieldset className="rounded-xs border border-control-border p-3">
                      <legend className="px-1 text-xs font-medium text-control-light">
                        {t("data-source.ssl.client-identity")}
                      </legend>
                      <div className="flex flex-col gap-y-3">
                        {showClientCertSourceUi && (
                          <div className="flex flex-col gap-y-1">
                            <label className="textlabel block">
                              {t("data-source.ssl.client-cert-source.self")}
                            </label>
                            <ClientCertSourceSelector
                              value={
                                resolvedClientCertSource ===
                                LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
                                  ? LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM
                                  : resolvedClientCertSource
                              }
                              onChange={onClientCertSourceChange}
                              disabled={disabled || !canSelectMutualTls}
                              isSaaSMode={isSaaSMode}
                            />
                          </div>
                        )}
                        {renderClientCertMaterial()}
                      </div>
                    </fieldset>
                  )}
                </>
              ) : (
                // existing non-posture branch
              )}
```

Update `renderCaMaterial` so file path inputs are disabled in SaaS mode:

```tsx
            disabled={disabled || isSaaSMode}
```

Update `renderClientCertMaterial` so file path inputs are disabled in SaaS mode:

```tsx
              disabled={disabled || isSaaSMode}
```

- [ ] **Step 7: Preserve legacy Vault behavior**

Make sure the `!showPerGroupSourceUi ? renderLegacyMaterial() : ...` branch remains available when `showPostureUi` is false. Vault calls `SslCertificateForm` without `useSsl`, `posture`, or source props; it should keep rendering its existing tabs/textareas.

- [ ] **Step 8: Run component tests**

Run:

```bash
pnpm --dir frontend test -- SslCertificateForm
```

Expected: PASS.

- [ ] **Step 9: Commit form redesign**

Run:

```bash
git add frontend/src/react/components/instance/SslCertificateForm.tsx frontend/src/react/components/instance/SslCertificateForm.test.tsx
git commit -m "feat: redesign TLS certificate form around posture"
```

---

### Task 5: Wire Posture and SaaS Mode in Data Source Form

**Files:**
- Modify: `frontend/src/react/components/instance/DataSourceForm.tsx`

- [ ] **Step 1: Update imports**

In `frontend/src/react/components/instance/DataSourceForm.tsx`, change:

```ts
import { useSubscriptionV1Store } from "@/store";
```

to:

```ts
import { useActuatorV1Store, useSubscriptionV1Store } from "@/store";
```

Extend the TLS imports:

```ts
import {
  applyLocalTlsCaSource,
  applyLocalTlsClientCertSource,
  applyLocalTlsPosture,
  disableLocalTls,
  getLocalTlsCaSource,
  getLocalTlsClientCertSource,
  getLocalTlsPosture,
  LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM,
  LOCAL_TLS_CLIENT_CERT_SOURCE_NONE,
  LOCAL_TLS_POSTURE_DISABLED,
  LOCAL_TLS_POSTURE_MUTUAL_TLS,
  LOCAL_TLS_POSTURE_TLS,
  type LocalTlsPosture,
} from "./tls";
```

- [ ] **Step 2: Add SaaS and posture state**

After the subscription store line:

```ts
  const actuatorStore = useActuatorV1Store();
  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
```

After `localTlsClientCertSource` state:

```ts
  const [localTlsPosture, setLocalTlsPosture] = useState(
    getLocalTlsPosture(dataSource)
  );
```

In the TLS sync effect, set posture when the data source changes or when `updateSsl` is not set:

```ts
      setLocalTlsPosture(getLocalTlsPosture(dataSource));
```

Add these dependencies to the effect only if not already present:

```ts
    dataSource.sslCertPathSet,
    dataSource.sslKeyPathSet,
```

- [ ] **Step 3: Add posture change handler**

Before the JSX return, add:

```ts
  const onLocalTlsPostureChange = useCallback(
    (posture: LocalTlsPosture) => {
      setLocalTlsPosture(posture);
      if (posture === LOCAL_TLS_POSTURE_DISABLED) {
        setLocalTlsCaSource("SYSTEM_TRUST");
        setLocalTlsClientCertSource(LOCAL_TLS_CLIENT_CERT_SOURCE_NONE);
        update({ ...disableLocalTls(dataSource), updateSsl: true });
        return;
      }

      const next = applyLocalTlsPosture(dataSource, posture);
      const enablingTls = !dataSource.useSsl;
      if (posture === LOCAL_TLS_POSTURE_TLS) {
        setLocalTlsClientCertSource(LOCAL_TLS_CLIENT_CERT_SOURCE_NONE);
      }
      if (
        posture === LOCAL_TLS_POSTURE_MUTUAL_TLS &&
        localTlsClientCertSource === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
      ) {
        setLocalTlsClientCertSource(LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM);
      }

      update({
        ...next,
        verifyTlsCertificate: enablingTls
          ? true
          : dataSource.verifyTlsCertificate,
        updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
          useSsl: true,
          clientCert: posture === LOCAL_TLS_POSTURE_TLS,
        }),
      });
    },
    [dataSource, localTlsClientCertSource, update]
  );
```

- [ ] **Step 4: Replace the old SSL section label and props**

In the SSL section, replace the visible label text:

```tsx
{t("data-source.ssl-connection")}
```

with:

```tsx
{t("data-source.ssl.connection-security")}
```

In the `SslCertificateForm` call, add:

```tsx
                posture={localTlsPosture}
                onPostureChange={onLocalTlsPostureChange}
                isSaaSMode={isSaaSMode}
```

Keep `useSsl` and `onUseSslChange` for compatibility during the transition, but `SslCertificateForm` should hide the switch when posture props are present.

- [ ] **Step 5: Preserve source change behavior**

When CA source changes, keep the existing handler.

When client source changes, keep the existing handler, but make sure SaaS disabled controls prevent choosing `FILE_PATH` before this handler runs. The handler should still tolerate existing `FILE_PATH` state because saved file-path rows must render truthfully.

- [ ] **Step 6: Run targeted tests**

Run:

```bash
pnpm --dir frontend test -- SslCertificateForm common
```

Expected: PASS.

- [ ] **Step 7: Commit DataSourceForm wiring**

Run:

```bash
git add frontend/src/react/components/instance/DataSourceForm.tsx
git commit -m "feat: wire TLS posture into data source form"
```

---

### Task 6: Polish, Type Check, and Final Verification

**Files:**
- Modify only files touched in earlier tasks if verification finds issues.

- [ ] **Step 1: Run frontend fixer**

Run:

```bash
pnpm --dir frontend fix
```

Expected: command completes and may modify formatting/import ordering.

- [ ] **Step 2: Run frontend checks**

Run:

```bash
pnpm --dir frontend check
```

Expected: PASS.

- [ ] **Step 3: Run React type check**

Run:

```bash
pnpm --dir frontend type-check
```

Expected: PASS.

- [ ] **Step 4: Run frontend tests**

Run:

```bash
pnpm --dir frontend test
```

Expected: PASS.

- [ ] **Step 5: Review final diff**

Run:

```bash
git diff --stat HEAD
git diff HEAD -- frontend/src/react/components/instance frontend/src/react/components/ui frontend/src/react/locales
```

Expected: diff only contains the planned TLS posture UI, helper, segmented control, tests, and locale changes.

- [ ] **Step 6: Commit verification fixes**

If `pnpm --dir frontend fix` or manual verification fixes changed files, commit them:

```bash
git add frontend/src/react/components/instance frontend/src/react/components/ui frontend/src/react/locales
git commit -m "chore: polish TLS posture UI"
```

If there are no changes, skip this commit.
