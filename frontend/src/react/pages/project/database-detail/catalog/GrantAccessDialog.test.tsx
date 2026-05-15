import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SensitiveColumn } from "@/react/lib/sensitive-data/types";

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
    useVueState: vi.fn((getter: () => unknown) => getter()),
    usePolicyV1Store: vi.fn(() => ({
      getOrFetchPolicyByParentAndType: vi.fn(),
      upsertPolicy: vi.fn(),
    })),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/react/lib/sensitive-data/components-utils", () => ({
  getClassificationLevelOptions: () => [],
}));

vi.mock("@/react/lib/sensitive-data/exemptionDataUtils", () => ({
  rewriteResourceDatabase: vi.fn((expression: string) => expression),
}));

vi.mock("@/react/lib/sensitive-data/utils", () => ({
  convertSensitiveColumnToDatabaseResource: vi.fn((column) => ({
    databaseFullName: column.database.name,
    schema: column.maskData.schema,
    table: column.maskData.table,
    columns: [column.maskData.column].filter(Boolean),
  })),
  getExpressionsForDatabaseResource: vi.fn(() => []),
}));

vi.mock("@/plugins/cel", () => ({
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

vi.mock("@/react/components/AccountMultiSelect", () => ({
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

vi.mock("@/react/components/DatabaseResourceSelector", () => ({
  DatabaseResourceSelector: ({
    value,
    includeColumns,
  }: {
    value: unknown;
    includeColumns?: boolean;
  }) => (
    <div
      data-include-columns={String(includeColumns)}
      data-testid="database-resource-selector"
    >
      {JSON.stringify(value)}
    </div>
  ),
}));

vi.mock("@/react/components/ExprEditor", () => ({
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

vi.mock("@/react/components/FeatureAttention", () => ({
  FeatureAttention: () => <div data-testid="feature-attention" />,
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: () => <div data-testid="feature-badge" />,
}));

vi.mock("@/react/components/ui/feature-modal", () => ({
  FeatureModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="feature-modal" /> : null,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: (props: ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/react/components/ui/expiration-picker", () => ({
  ExpirationPicker: ({ minDate }: { minDate: string }) => (
    <div data-min-date={minDate} data-testid="expiration-picker" />
  ),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  featureToRef: mocks.featureToRef,
  pushNotification: vi.fn(),
  usePolicyV1Store: mocks.usePolicyV1Store,
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

import { GrantAccessDialog } from "./GrantAccessDialog";

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

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

    const radioList = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[type="radio"]')
    );

    expect(radioList).toHaveLength(3);
    expect(radioList[2]?.checked).toBe(true);

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

    const radioList = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[type="radio"]')
    );
    expect(radioList[1]?.checked).toBe(true);
    expect(radioList[2]?.disabled).toBe(false);

    const exprEditor = container.querySelector('[data-testid="expr-editor"]');
    expect(exprEditor?.textContent).toContain("serialized-selection");

    unmount();
  });

  test("allows select mode after CEL scope becomes column-scoped", async () => {
    const { container, unmount } = renderGrantAccessDialog();
    await flush();

    let radioList = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[type="radio"]')
    );
    expect(radioList[2]?.checked).toBe(true);
    expect(radioList[2]?.disabled).toBe(false);

    await click(radioList[1]!);
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="expr-editor-set-column-scope"]'
      )!
    );

    radioList = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[type="radio"]')
    );
    expect(radioList[1]?.checked).toBe(true);
    expect(radioList[2]?.disabled).toBe(false);

    await click(radioList[2]!);

    radioList = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[type="radio"]')
    );
    expect(radioList[2]?.checked).toBe(true);
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

  test("keeps the latest mode when an earlier async conversion resolves late", async () => {
    const pendingConversion = deferred<[{ parsed: true }]>();
    mocks.batchConvertCELStringToParsedExpr.mockImplementationOnce(
      () => pendingConversion.promise
    );

    const { container, unmount } = renderGrantAccessDialog();
    await flush();

    const radioList = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[type="radio"]')
    );

    expect(radioList[2]?.checked).toBe(true);

    act(() => {
      radioList[1]?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });
    await flush();

    act(() => {
      radioList[0]?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });
    await flush();

    pendingConversion.resolve([{ parsed: true }]);
    await flush();

    const refreshedRadioList = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[type="radio"]')
    );
    expect(refreshedRadioList[0]?.checked).toBe(true);
    expect(refreshedRadioList[1]?.checked).toBe(false);
    expect(
      container.querySelector('[data-testid="expr-editor"]')
    ).not.toBeTruthy();

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
