import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { SensitiveColumn } from "@/lib/sensitive-data/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const convertedExpr = {
    type: "ConditionGroup",
    operator: "_&&_",
    args: [
      {
        type: "Condition",
        operator: "_==_",
        args: ["resource.database", "instances/inst1/databases/db1"],
      },
    ],
  };

  return {
    convertedExpr,
    batchConvertCELStringToParsedExpr: vi.fn(async () => [{ parsed: true }]),
    resolveCELExpr: vi.fn(() => convertedExpr),
    stringifyConditionExpression: vi.fn(() => "serialized-selection"),
    featureToRef: vi.fn(() => ({ value: true })),
    getOrFetchPolicyByParentAndType: vi.fn(),
    upsertPolicy: vi.fn(),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/lib/sensitive-data/components-utils", () => ({
  getClassificationLevelOptions: () => [],
}));

vi.mock("@/lib/sensitive-data/exemptionDataUtils", () => ({
  rewriteResourceDatabase: vi.fn((expression: string) => expression),
}));

vi.mock("@/lib/sensitive-data/utils", () => ({
  convertSensitiveColumnToDatabaseResource: vi.fn((column) => ({
    databaseFullName: column.database.name,
    schema: column.maskData.schema,
    table: column.maskData.table,
    columns: [column.maskData.column].filter(Boolean),
  })),
  getExpressionsForDatabaseResource: vi.fn(() => []),
}));

vi.mock("@/modules/cel", () => ({
  ExprType: {
    Condition: "Condition",
    ConditionGroup: "ConditionGroup",
    RawString: "RawString",
  },
  buildCELExpr: vi.fn(),
  emptySimpleExpr: vi.fn(() => ({
    type: "ConditionGroup",
    operator: "_&&_",
    args: [],
  })),
  isConditionExpr: vi.fn((expr) => expr?.type === "Condition"),
  isConditionGroupExpr: vi.fn((expr) => expr?.type === "ConditionGroup"),
  isRawStringExpr: vi.fn((expr) => expr?.type === "RawString"),
  resolveCELExpr: mocks.resolveCELExpr,
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
    value,
    includeColumns,
    onChange,
  }: {
    value: unknown;
    includeColumns?: boolean;
    onChange?: (resources: unknown[]) => void;
  }) => (
    <div
      data-include-columns={String(includeColumns)}
      data-testid="database-resource-selector"
    >
      {JSON.stringify(value)}
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
    </div>
  ),
}));

vi.mock("@/components/ExprEditor", () => ({
  ExprEditor: ({
    expr,
    onUpdate,
  }: {
    expr: unknown;
    onUpdate: (expr: unknown) => void;
  }) => (
    <div>
      <div data-testid="expr-editor">{JSON.stringify(expr)}</div>
      <button
        data-testid="expr-editor-set-column-scope"
        onClick={() =>
          onUpdate({
            type: "ConditionGroup",
            operator: "_&&_",
            args: [
              {
                type: "Condition",
                operator: "_==_",
                args: ["resource.column_name", "id"],
              },
            ],
          })
        }
        type="button"
      />
    </div>
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

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/components/ui/expiration-picker", () => ({
  ExpirationPicker: ({
    minDate,
    onChange,
  }: {
    minDate: string;
    onChange?: (value: string) => void;
  }) => (
    <div data-min-date={minDate} data-testid="expiration-picker">
      <button
        data-testid="set-expiration"
        type="button"
        onClick={() => onChange?.("2026-06-02T10:00:00.000Z")}
      />
    </div>
  ),
}));

vi.mock("@/components/ui/input", () => ({
  Input: (props: InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/stores", () => ({
  featureToRef: mocks.featureToRef,
  pushNotification: vi.fn(),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (
      selector: (state: {
        settingsByName: Record<string, unknown>;
        hasInstanceFeature: () => boolean;
      }) => unknown
    ) =>
      selector({
        settingsByName: {},
        hasInstanceFeature: () => mocks.featureToRef().value,
      }),
    {
      getState: () => ({
        getOrFetchPolicyByParentAndType: mocks.getOrFetchPolicyByParentAndType,
        upsertPolicy: mocks.upsertPolicy,
        classification: () => [],
        hasInstanceFeature: () => mocks.featureToRef().value,
      }),
    }
  ),
}));

vi.mock("@/utils", () => ({
  batchConvertParsedExprToCELString: vi.fn(),
  getDatabaseNameOptionConfig: vi.fn(() => ({ options: [] })),
}));

vi.mock("@/utils/issue/cel", () => ({
  convertFromExpr: vi.fn(() => ({
    databaseResources: [],
  })),
  stringifyConditionExpression: mocks.stringifyConditionExpression,
}));

vi.mock("@/utils/v1/cel", () => ({
  batchConvertCELStringToParsedExpr: mocks.batchConvertCELStringToParsedExpr,
}));

import { getExpressionsForDatabaseResource } from "@/lib/sensitive-data/utils";
import { GrantAccessDialog } from "./GrantAccessDialog";

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

const getRadios = (container: HTMLElement) =>
  Array.from(container.querySelectorAll<HTMLElement>('[role="radio"]'));

const isChecked = (radio: HTMLElement | undefined) =>
  radio?.getAttribute("aria-checked") === "true";

const isDisabled = (radio: HTMLElement | undefined) =>
  radio?.getAttribute("aria-disabled") === "true";

const deferred = <T,>() => {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((resolver) => {
    resolve = resolver;
  });
  return { promise, resolve };
};

const click = async (element: HTMLElement) => {
  act(() => {
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
  await flush();
};

const changeTextInput = async (input: HTMLInputElement, value: string) => {
  act(() => {
    const descriptor = Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    );
    descriptor?.set?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
    input.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await flush();
};

const createColumnList = (): SensitiveColumn[] =>
  [
    {
      database: {
        name: "instances/inst1/databases/db1",
      } as SensitiveColumn["database"],
      maskData: {
        schema: "public",
        table: "book",
        column: "",
        semanticTypeId: "",
        classificationId: "",
        target: {} as SensitiveColumn["maskData"]["target"],
      } as SensitiveColumn["maskData"],
    },
  ] satisfies SensitiveColumn[];

const renderGrantAccessDialog = ({
  open = true,
  columnList = createColumnList(),
}: {
  open?: boolean;
  columnList?: SensitiveColumn[];
} = {}) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  const render = (nextOpen = open, nextColumnList = columnList) => {
    act(() => {
      root.render(
        <GrantAccessDialog
          open={nextOpen}
          projectName="projects/proj1"
          columnList={nextColumnList}
          onDismiss={vi.fn()}
        />
      );
    });
  };

  render(open);

  return {
    container,
    render,
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

describe("GrantAccessDialog", () => {
  beforeEach(() => {
    mocks.batchConvertCELStringToParsedExpr.mockClear();
    mocks.resolveCELExpr.mockClear();
    mocks.stringifyConditionExpression.mockClear();
  });

  test("preserves selected scope when switching from select to expression mode", async () => {
    const { container, unmount } = renderGrantAccessDialog();
    await flush();

    const radioList = getRadios(container);

    expect(radioList).toHaveLength(3);
    expect(isChecked(radioList[2])).toBe(true);

    await click(radioList[1]!);

    expect(mocks.stringifyConditionExpression).toHaveBeenCalledWith({
      databaseResources: [
        {
          columns: [],
          databaseFullName: "instances/inst1/databases/db1",
          schema: "public",
          table: "book",
        },
      ],
    });
    expect(mocks.batchConvertCELStringToParsedExpr).toHaveBeenCalledWith([
      "serialized-selection",
    ]);

    const exprEditor = container.querySelector('[data-testid="expr-editor"]');
    expect(exprEditor?.textContent).toContain(
      JSON.stringify(mocks.convertedExpr)
    );

    unmount();
  });

  test("falls back to a raw expression when initial CEL conversion fails", async () => {
    mocks.batchConvertCELStringToParsedExpr.mockImplementationOnce(
      async () => [undefined] as unknown as [{ parsed: true }]
    );

    const { container, unmount } = renderGrantAccessDialog({
      columnList: [
        {
          database: {
            name: "instances/inst1/databases/db1",
          } as SensitiveColumn["database"],
          maskData: {
            schema: "public",
            table: "book",
            column: "id",
            semanticTypeId: "",
            classificationId: "",
            target: {} as SensitiveColumn["maskData"]["target"],
          } as SensitiveColumn["maskData"],
        },
      ],
    });
    await flush();

    const radioList = getRadios(container);
    expect(isChecked(radioList[1])).toBe(true);
    expect(isDisabled(radioList[2])).toBe(false);

    const exprEditor = container.querySelector('[data-testid="expr-editor"]');
    expect(exprEditor?.textContent).toContain("serialized-selection");

    unmount();
  });

  test("allows select mode after CEL scope becomes column-scoped", async () => {
    const { container, unmount } = renderGrantAccessDialog();
    await flush();

    let radioList = getRadios(container);
    expect(isChecked(radioList[2])).toBe(true);
    expect(isDisabled(radioList[2])).toBe(false);

    await click(radioList[1]!);
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="expr-editor-set-column-scope"]'
      )!
    );

    radioList = getRadios(container);
    expect(isChecked(radioList[1])).toBe(true);
    expect(isDisabled(radioList[2])).toBe(false);

    await click(radioList[2]!);

    radioList = getRadios(container);
    expect(isChecked(radioList[2])).toBe(true);
    expect(
      container
        .querySelector('[data-testid="database-resource-selector"]')
        ?.getAttribute("data-include-columns")
    ).toBe("true");

    unmount();
  });

  test("refreshes the expiration minimum when reopening the dialog", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-14T08:00:00Z"));

    const { container, render, unmount } = renderGrantAccessDialog();
    await flush();

    const initialPicker = container.querySelector(
      '[data-testid="expiration-picker"]'
    );
    expect(initialPicker?.getAttribute("data-min-date")).toBe(
      "2026-04-14T00:00"
    );

    vi.setSystemTime(new Date("2026-04-15T08:00:00Z"));
    render(false);
    await flush();
    render(true);
    await flush();

    const reopenedPicker = container.querySelector(
      '[data-testid="expiration-picker"]'
    );
    expect(reopenedPicker?.getAttribute("data-min-date")).toBe(
      "2026-04-15T00:00"
    );

    unmount();
    vi.useRealTimers();
  });

  test("disables mode changes while async conversion is pending", async () => {
    const pendingConversion = deferred<[{ parsed: true }]>();
    mocks.batchConvertCELStringToParsedExpr.mockImplementationOnce(
      () => pendingConversion.promise
    );

    const { container, unmount } = renderGrantAccessDialog();
    await flush();

    const radioList = getRadios(container);

    expect(isChecked(radioList[2])).toBe(true);

    await click(radioList[1]!);
    await flush();

    const pendingRadioList = getRadios(container);
    expect(isDisabled(pendingRadioList[0])).toBe(true);
    expect(isDisabled(pendingRadioList[1])).toBe(true);
    expect(isDisabled(pendingRadioList[2])).toBe(true);

    pendingConversion.resolve([{ parsed: true }]);
    await flush();

    const refreshedRadioList = getRadios(container);
    expect(isChecked(refreshedRadioList[1])).toBe(true);
    expect(isChecked(refreshedRadioList[2])).toBe(false);
    expect(container.querySelector('[data-testid="expr-editor"]')).toBeTruthy();

    unmount();
  });

  test("does not reset form state when reopened props are semantically unchanged", async () => {
    const initialColumnList = createColumnList();
    const { container, render, unmount } = renderGrantAccessDialog({
      columnList: initialColumnList,
    });
    await flush();

    const descriptionInput = container.querySelector<HTMLInputElement>(
      'input[placeholder="common.description"]'
    );
    expect(descriptionInput).toBeTruthy();

    await changeTextInput(descriptionInput!, "temporary reason");
    expect(descriptionInput?.value).toBe("temporary reason");

    render(true, createColumnList());
    await flush();

    const refreshedDescriptionInput = container.querySelector<HTMLInputElement>(
      'input[placeholder="common.description"]'
    );
    expect(refreshedDescriptionInput?.value).toBe("temporary reason");

    unmount();
  });
});

// ============================================================
// BYT-9791 — REGRESSION GUARD (bug fixed by #20687).
//
// This was filed as a sibling of BYT-9788: GrantAccessDialog used to emit the
// unwrapped `(c1) || (c2) && request.time < …`, which cel-go reads as
// `(c1) || ((c2) && time)`, leaking the first resource past expiry. #20687
// extracted the shared buildMaskingExemption helper and routed both this dialog
// and the create page through it, so the OR-group is now wrapped and the time
// constraint binds to every resource. This test drives the real dialog and pins
// the corrected output so GrantAccessDialog can't regress to the buggy shape.
// ============================================================

describe("GrantAccessDialog — composite exemption precedence (BYT-9791)", () => {
  const c1 =
    'resource.instance_id == "inst1" && resource.database_name == "db1" && resource.table_name == "t1"';
  const c2 =
    'resource.instance_id == "inst1" && resource.database_name == "db1" && resource.table_name == "t2"';
  const timeClause = 'request.time < timestamp("2026-06-02T10:00:00.000Z")';

  beforeEach(() => {
    mocks.upsertPolicy.mockClear();
    mocks.getOrFetchPolicyByParentAndType.mockReset();
    // Return real per-resource clauses so onSubmit assembles a real expression.
    vi.mocked(getExpressionsForDatabaseResource).mockImplementation(
      (resource: { table?: string }) => [
        'resource.instance_id == "inst1"',
        'resource.database_name == "db1"',
        `resource.table_name == "${resource.table}"`,
      ]
    );
  });

  afterEach(() => {
    // Restore the inert default the other tests rely on.
    vi.mocked(getExpressionsForDatabaseResource).mockReturnValue([]);
  });

  test("wraps the OR-group so the expiration binds to every resource", async () => {
    const { container, unmount } = renderGrantAccessDialog();
    await flush();

    // Default mode for a non-column-scoped selection is SELECT (radio index 2).
    const radios = getRadios(container);
    expect(isChecked(radios[2])).toBe(true);

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

    const confirm = Array.from(
      container.querySelectorAll<HTMLButtonElement>("button")
    ).find((b) => b.textContent === "common.confirm");
    expect(confirm).toBeTruthy();
    expect(confirm?.disabled).toBe(false);

    await click(confirm!);
    await flush();

    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(1);
    const arg = mocks.upsertPolicy.mock.calls[0][0] as {
      policy: {
        policy: {
          value: { exemptions: { condition?: { expression: string } }[] };
        };
      };
    };
    const expression =
      arg.policy.policy.value.exemptions[0]?.condition?.expression;

    // FIXED (#20687): the OR-group is wrapped and the time constraint leads, so
    // the expiry binds to every resource — not just the last branch.
    expect(expression).toBe(`${timeClause} && ((${c1}) || (${c2}))`);
    expect(expression).not.toBe(`(${c1}) || (${c2}) && ${timeClause}`);

    unmount();
  });
});
