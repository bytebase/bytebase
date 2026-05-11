import { act, type ReactElement, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import type { Factor } from "@/plugins/cel/types/factor";
import { type ConditionGroupExpr, ExprType } from "@/plugins/cel/types/simple";
import {
  CEL_ATTRIBUTE_RISK_LEVEL,
  CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
} from "@/utils/cel-attributes";
import { ExprEditor, type OptionConfig } from "./ExprEditor";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

class ResizeObserverStub implements ResizeObserver {
  constructor(_callback: ResizeObserverCallback) {}
  observe() {}
  unobserve() {}
  disconnect() {}
}

globalThis.ResizeObserver = ResizeObserverStub;

const { MockExprType } = vi.hoisted(() => ({
  MockExprType: {
    RawString: "RawString",
    Condition: "Condition",
    ConditionGroup: "ConditionGroup",
  } as const,
}));

vi.mock("@/plugins/cel", () => ({
  ExprType: MockExprType,
  LogicalOperatorList: ["_&&_", "_||_"],
  getOperatorListByFactor: () => ["_==_", "@in"],
  isBooleanFactor: () => false,
  isCollectionOperator: (operator: string) =>
    operator === "@in" || operator === "@not_in",
  isConditionExpr: (expr: { type: string }) =>
    expr.type === MockExprType.Condition,
  isConditionGroupExpr: (expr: { type: string }) =>
    expr.type === MockExprType.ConditionGroup,
  isNumberFactor: () => false,
  isRawStringExpr: (expr: { type: string }) =>
    expr.type === MockExprType.RawString,
  isStringFactor: () => true,
  isStringOperator: (operator: string) =>
    ["contains", "@not_contains", "matches", "startsWith", "endsWith"].includes(
      operator
    ),
  isTimestampFactor: () => false,
  operatorDisplayLabel: (operator: string) => operator,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/utils", () => ({
  getDefaultPagination: () => ({
    pageSize: 1000,
    pageToken: "",
  }),
}));

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) => (
    <span>{environmentName}</span>
  ),
}));

vi.mock("@/store/modules/v1/common", () => ({
  environmentNamePrefix: "environments/",
}));

const optionConfigMap = new Map<Factor, OptionConfig>([
  [
    CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
    {
      options: [
        { value: "DDL", label: "DDL" },
        { value: "DML", label: "DML" },
        { value: "DCL", label: "DCL" },
      ],
    },
  ],
  [
    CEL_ATTRIBUTE_RISK_LEVEL,
    {
      options: [
        { value: "HIGH", label: "High" },
        { value: "LOW", label: "Low" },
      ],
    },
  ],
]);

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  act(() => {
    root.render(element);
  });

  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

const flushEffects = async () => {
  await act(async () => {
    await Promise.resolve();
  });
};

describe("ExprEditor", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("keeps the next condition value after deleting the first condition", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_STATEMENT_SQL_TYPE, ["DDL"]],
        },
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RISK_LEVEL, ["HIGH"]],
        },
      ],
    };
    let currentExpr = initialExpr;

    const Harness = () => {
      const [expr, setExpr] = useState(initialExpr);
      return (
        <ExprEditor
          expr={expr}
          factorList={[
            CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
            CEL_ATTRIBUTE_RISK_LEVEL,
          ]}
          optionConfigMap={optionConfigMap}
          onUpdate={(next) => {
            currentExpr = next;
            setExpr(next);
          }}
        />
      );
    };

    const { container, unmount } = renderIntoContainer(<Harness />);
    await flushEffects();

    const deleteButtons = Array.from(
      container.querySelectorAll('button[type="button"]')
    ).filter((button) => button.className.includes("size-7"));
    expect(deleteButtons).toHaveLength(2);

    await act(async () => {
      deleteButtons[0]?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    await flushEffects();

    expect(currentExpr.args).toHaveLength(1);
    expect(currentExpr.args[0]).toMatchObject({
      type: ExprType.Condition,
      operator: "@in",
      args: [CEL_ATTRIBUTE_RISK_LEVEL, ["HIGH"]],
    });

    unmount();
  });

  test("static multi-select dropdown constrains overflow", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_STATEMENT_SQL_TYPE, []],
        },
      ],
    };

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_STATEMENT_SQL_TYPE]}
        optionConfigMap={optionConfigMap}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const trigger = Array.from(
      container.querySelectorAll('button[type="button"]')
    ).find((button) =>
      button.textContent?.includes("cel.condition.select-value")
    );
    expect(trigger).toBeInstanceOf(HTMLButtonElement);

    await act(async () => {
      trigger?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const allOption = Array.from(document.body.querySelectorAll("label")).find(
      (label) => label.textContent?.includes("common.all")
    );
    const dropdown = allOption?.parentElement;
    expect(dropdown).toBeInstanceOf(HTMLDivElement);
    expect(dropdown?.className).toContain("max-h-48");
    expect(dropdown?.className).toContain("overflow-y-auto");

    unmount();
  });
});
