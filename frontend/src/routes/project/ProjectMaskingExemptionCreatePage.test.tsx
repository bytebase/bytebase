import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// ============================================================
// BYT-9788 — REGRESSION test for PR #20683.
//
// Drives the REAL create-page builder (onSubmit) and asserts that the generated
// masking-exemption CEL wraps the whole resource OR-group BEFORE the expiration
// is AND-joined, so `request.time` binds to every resource:
//   request.time < timestamp("…") && ((c1) || (c2))
//
// The pre-fix builder produced `(c1) || (c2) && request.time < …`, which cel-go
// reads as `(c1) || ((c2) && time)` — the first resource never expired.
// (#20687 later shared this builder with GrantAccessDialog, fixing BYT-9791.)
// ============================================================

const mocks = vi.hoisted(() => ({
  upsertPolicy: vi.fn(),
  getOrFetchPolicyByParentAndType: vi.fn(),
  getOrFetchSettingByName: vi.fn(),
  classification: vi.fn(() => []),
  buildCELExpr: vi.fn(),
  batchConvertParsedExprToCELString: vi.fn(),
  getExpressionsForDatabaseResource: vi.fn(
    (_resource: { table?: string }): string[] => []
  ),
  routerBack: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/modules/cel", () => ({
  buildCELExpr: mocks.buildCELExpr,
  emptySimpleExpr: vi.fn(() => ({ type: "ConditionGroup", args: [] })),
  validateSimpleExpr: vi.fn(() => true),
  wrapAsGroup: vi.fn((expr) => expr),
}));

vi.mock("@/components/AccountMultiSelect", () => ({
  AccountMultiSelect: ({
    value,
    onChange,
  }: {
    value: string[];
    onChange: (value: string[]) => void;
  }) => (
    <button
      data-testid="account-multi-select"
      onClick={() => onChange([...value, "users/alice"])}
      type="button"
    />
  ),
}));

vi.mock("@/components/DatabaseResourceSelector", () => ({
  DatabaseResourceSelector: ({
    onChange,
  }: {
    onChange?: (resources: unknown[]) => void;
  }) => (
    <button
      data-testid="set-two-resources"
      type="button"
      onClick={() =>
        onChange?.([
          {
            databaseFullName: "instances/inst1/databases/db1",
            schema: "public",
            table: "t1",
          },
          {
            databaseFullName: "instances/inst1/databases/db1",
            schema: "public",
            table: "t2",
          },
        ])
      }
    />
  ),
}));

vi.mock("@/components/ExprEditor", () => ({
  ExprEditor: ({ onUpdate }: { onUpdate: (expr: unknown) => void }) => (
    <button
      data-testid="set-expr"
      type="button"
      onClick={() => onUpdate({ type: "ConditionGroup", args: ["set"] })}
    />
  ),
}));

vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => <div data-testid="feature-attention" />,
}));

vi.mock("@/components/FeatureBadge", () => ({
  FeatureBadge: () => <div data-testid="feature-badge" />,
}));

vi.mock("@/components/ui/feature-modal", () => ({
  FeatureModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="feature-modal" /> : null,
}));

vi.mock("@/components/ui/button", () => ({
  Button: (props: ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
}));

vi.mock("@/components/ui/expiration-picker", () => ({
  ExpirationPicker: ({ onChange }: { onChange?: (value: string) => void }) => (
    <button
      data-testid="set-expiration"
      type="button"
      onClick={() => onChange?.("2026-06-02T10:00:00.000Z")}
    />
  ),
}));

vi.mock("@/components/ui/input", () => ({
  Input: (props: InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/proj1" }),
}));

vi.mock("@/lib/sensitive-data/components-utils", () => ({
  getClassificationLevelOptions: () => [],
}));

vi.mock("@/lib/sensitive-data/exemptionDataUtils", () => ({
  rewriteResourceDatabase: vi.fn((expression: string) => expression),
}));

vi.mock("@/lib/sensitive-data/utils", () => ({
  getExpressionsForDatabaseResource: mocks.getExpressionsForDatabaseResource,
}));

vi.mock("@/app/router", () => ({
  router: { back: mocks.routerBack },
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/stores/modules/v1/common", () => ({
  projectNamePrefix: "projects/",
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (
      selector: (state: {
        projectsByName: Record<string, unknown>;
        settingsByName: Record<string, unknown>;
        hasFeature: () => boolean;
      }) => unknown
    ) =>
      selector({
        projectsByName: {},
        settingsByName: {},
        hasFeature: () => true,
      }),
    {
      getState: () => ({
        getOrFetchSettingByName: mocks.getOrFetchSettingByName,
        getOrFetchPolicyByParentAndType: mocks.getOrFetchPolicyByParentAndType,
        upsertPolicy: mocks.upsertPolicy,
        classification: mocks.classification,
      }),
    }
  ),
}));

vi.mock("@/utils", () => ({
  batchConvertParsedExprToCELString: mocks.batchConvertParsedExprToCELString,
  getDatabaseNameOptionConfig: vi.fn(() => ({ options: [] })),
}));

vi.mock("@/utils/cel-attributes", () => ({
  CEL_ATTRIBUTE_RESOURCE_DATABASE: "resource.database",
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME: "resource.table_name",
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME: "resource.schema_name",
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME: "resource.column_name",
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL: "resource.classification_level",
}));

import { ProjectMaskingExemptionCreatePage } from "./ProjectMaskingExemptionCreatePage";

const TIME = 'request.time < timestamp("2026-06-02T10:00:00.000Z")';
const c1 =
  'resource.instance_id == "inst1" && resource.database_name == "db1" && resource.table_name == "t1"';
const c2 =
  'resource.instance_id == "inst1" && resource.database_name == "db1" && resource.table_name == "t2"';

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

const click = async (element: HTMLElement) => {
  act(() => {
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
  await flush();
};

const render = () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  act(() => {
    root.render(<ProjectMaskingExemptionCreatePage projectId="proj1" />);
  });
  return {
    container,
    unmount: () => act(() => root.unmount()),
  };
};

const submittedExpression = (): string | undefined => {
  const arg = mocks.upsertPolicy.mock.calls[0]?.[0] as
    | {
        policy: {
          policy: {
            value: { exemptions: { condition?: { expression: string } }[] };
          };
        };
      }
    | undefined;
  return arg?.policy.policy.value.exemptions[0]?.condition?.expression;
};

const clickRadio = async (container: HTMLElement, index: number) => {
  const radios = Array.from(
    container.querySelectorAll<HTMLElement>('[role="radio"]')
  );
  await click(radios[index]!);
};

const clickConfirm = async (container: HTMLElement) => {
  const confirm = Array.from(
    container.querySelectorAll<HTMLButtonElement>("button")
  ).find((b) => b.textContent === "common.confirm");
  expect(confirm).toBeTruthy();
  expect(confirm?.disabled).toBe(false);
  await click(confirm!);
};

describe("ProjectMaskingExemptionCreatePage builder (BYT-9788)", () => {
  beforeEach(() => {
    mocks.upsertPolicy.mockClear();
    mocks.getOrFetchPolicyByParentAndType.mockReset();
    mocks.buildCELExpr.mockReset();
    mocks.batchConvertParsedExprToCELString.mockReset();
    mocks.getExpressionsForDatabaseResource.mockImplementation(
      (resource: { table?: string }) => [
        'resource.instance_id == "inst1"',
        'resource.database_name == "db1"',
        `resource.table_name == "${resource.table}"`,
      ]
    );
  });

  afterEach(() => {
    mocks.getExpressionsForDatabaseResource.mockReset();
  });

  test("SELECT mode: wraps the OR-group so the expiration binds to every resource", async () => {
    const { container, unmount } = render();
    await flush();

    await clickRadio(container, 2); // SELECT
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-two-resources"]')!
    );
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-expiration"]')!
    );
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="account-multi-select"]'
      )!
    );

    await clickConfirm(container);
    await flush();

    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(1);
    // FIXED form: the OR-group is parenthesized and the time constraint leads.
    expect(submittedExpression()).toBe(`${TIME} && ((${c1}) || (${c2}))`);

    unmount();
  });

  test("SELECT mode without expiration: OR-group only, still parenthesized", async () => {
    const { container, unmount } = render();
    await flush();

    await clickRadio(container, 2); // SELECT
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-two-resources"]')!
    );
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="account-multi-select"]'
      )!
    );

    await clickConfirm(container);
    await flush();

    expect(submittedExpression()).toBe(`((${c1}) || (${c2}))`);

    unmount();
  });

  test("EXPRESSION mode: a top-level || in custom CEL is wrapped and bound by the expiration", async () => {
    mocks.buildCELExpr.mockResolvedValue({ parsed: true });
    mocks.batchConvertParsedExprToCELString.mockResolvedValue([
      'resource.database_name == "a" || resource.database_name == "b"',
    ]);

    const { container, unmount } = render();
    await flush();

    await clickRadio(container, 1); // EXPRESSION
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-expr"]')!
    );
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-expiration"]')!
    );
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="account-multi-select"]'
      )!
    );

    await clickConfirm(container);
    await flush();

    expect(submittedExpression()).toBe(
      `${TIME} && (resource.database_name == "a" || resource.database_name == "b")`
    );

    unmount();
  });

  test("ALL mode with expiration: time-only expression, no resource group", async () => {
    const { container, unmount } = render();
    await flush();

    // Default mode is ALL.
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-expiration"]')!
    );
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="account-multi-select"]'
      )!
    );

    await clickConfirm(container);
    await flush();

    expect(submittedExpression()).toBe(TIME);

    unmount();
  });
});
