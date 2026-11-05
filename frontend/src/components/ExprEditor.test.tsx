import * as stylex from "@stylexjs/stylex";
import { fireEvent, waitFor } from "@testing-library/react";
import { act, type ReactElement, StrictMode, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import type { Factor } from "@/modules/cel/types/factor";
import { type ConditionGroupExpr, ExprType } from "@/modules/cel/types/simple";
import {
  applyAgentExpressionDraft,
  canApplyAgentExpressionDraft,
} from "@/modules/agent/logic/expressionDraft";
import {
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
  CEL_ATTRIBUTE_RISK_LEVEL,
  CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
} from "@/utils/cel-attributes";
import { ExprEditor, type OptionConfig } from "./ExprEditor";
import { controlSizeStyle } from "./ui/styles.stylex";

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

const expectClasses = (className: string | undefined, expected: string) => {
  for (const expectedClass of expected.split(" ")) {
    expect(className ?? "").toContain(expectedClass);
  }
};

const { MockExprType } = vi.hoisted(() => ({
  MockExprType: {
    RawString: "RawString",
    Condition: "Condition",
    ConditionGroup: "ConditionGroup",
  } as const,
}));

const agentMock = vi.hoisted(() => ({
  openAgentWithPrompt: vi.fn(
    (
      _prompt: string,
      _references: unknown,
      onChatCreated?: (chatId: string) => void
    ) => onChatCreated?.("expression-chat")
  ),
}));

const expressionMock = vi.hoisted(() => ({
  buildCELExpr: vi.fn(async () => ({
    exprKind: { case: "callExpr", value: {} },
  })),
  batchConvertCELStringToParsedExpr: vi.fn(async () => [
    { exprKind: { case: "callExpr", value: {} } },
  ]),
  batchConvertParsedExprToCELString: vi.fn(async () => ["canonical", "canonical"]),
}));

vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({
  openAgentWithPrompt: agentMock.openAgentWithPrompt,
}));

vi.mock("@/utils/v1/cel", () => ({
  batchConvertCELStringToParsedExpr:
    expressionMock.batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString:
    expressionMock.batchConvertParsedExprToCELString,
}));

vi.mock("@/modules/cel", () => ({
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
  buildCELExpr: expressionMock.buildCELExpr,
  resolveCELExpr: () => ({
    type: "ConditionGroup",
    operator: "_&&_",
    args: [],
  }),
  wrapAsGroup: (expr: unknown) => expr,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, string>) =>
      key === "cel.agent-help.starter"
        ? `${key}: ${options?.mode}; ${options?.attributes}`
        : options?.defaultValue ?? key,
  }),
}));

vi.mock("@/utils", () => ({
  getDefaultPagination: () => ({
    pageSize: 1000,
    pageToken: "",
  }),
}));

vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) => (
    <span>{environmentName}</span>
  ),
}));

vi.mock("@/stores/modules/v1/common", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/stores/modules/v1/common")>()),
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

const searchableOptionConfigMap = new Map<Factor, OptionConfig>([
  [
    CEL_ATTRIBUTE_RESOURCE_DATABASE,
    {
      options: [],
      search: async () => ({
        options: [],
        nextPageToken: "",
      }),
    },
  ],
  [
    CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
    {
      options: [],
      search: async () => ({
        options: [],
        nextPageToken: "",
      }),
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
    rerender: (element: ReactElement) =>
      act(() => {
        root.render(element);
      }),
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
    agentMock.openAgentWithPrompt.mockClear();
    expressionMock.buildCELExpr
      .mockReset()
      .mockResolvedValue({ exprKind: { case: "callExpr", value: {} } });
    expressionMock.batchConvertCELStringToParsedExpr
      .mockReset()
      .mockResolvedValue([{ exprKind: { case: "callExpr", value: {} } }]);
    expressionMock.batchConvertParsedExprToCELString
      .mockReset()
      .mockResolvedValue(["canonical", "canonical"]);
  });

  test("opens Agent help and applies a generated expression to the selector", async () => {
    const onUpdate = vi.fn();
    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={{ type: ExprType.ConditionGroup, operator: "_&&_", args: [] }}
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        onUpdate={onUpdate}
      />
    );

    const askAgent = container.querySelector<HTMLButtonElement>(
      'button[aria-label="Ask Agent to help build this expression"]'
    );
    expect(askAgent).toBeTruthy();
    expect(askAgent?.textContent).toBe("");
    expect(askAgent?.querySelector(".lucide-bot")).toBeTruthy();
    expect(askAgent?.className).toContain("h-7");
    await act(async () => {
      fireEvent.click(askAgent!);
    });
    expect(agentMock.openAgentWithPrompt).toHaveBeenCalledOnce();
    expect(agentMock.openAgentWithPrompt.mock.calls[0][0]).toContain(
      CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID
    );
    expect(agentMock.openAgentWithPrompt.mock.calls[0][0]).toContain(
      "cel.agent-help.starter"
    );
    expect(agentMock.openAgentWithPrompt.mock.calls[0][0]).toContain(
      CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID
    );

    await act(async () => {
      await applyAgentExpressionDraft(
        "expression-chat",
        'resource.environment == "prod"',
        "selector"
      );
    });
    expect(onUpdate).toHaveBeenCalledWith({
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [],
    });
    unmount();
  });

  test("does not apply a selector draft when conversion changes its meaning", async () => {
    const onUpdate = vi.fn();
    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={{ type: ExprType.ConditionGroup, operator: "_&&_", args: [] }}
        factorList={[CEL_ATTRIBUTE_RESOURCE_DATABASE]}
        onUpdate={onUpdate}
      />
    );
    const askAgent = container.querySelector<HTMLButtonElement>(
      'button[aria-label="Ask Agent to help build this expression"]'
    );
    expressionMock.batchConvertParsedExprToCELString.mockResolvedValueOnce([
      '!(resource.database_name == "prod")',
      'resource.database_name == "prod"',
    ]);

    let applied = true;
    await act(async () => {
      fireEvent.click(askAgent!);
      applied = await applyAgentExpressionDraft(
        "expression-chat",
        '!(resource.database_name == "prod")',
        "selector"
      );
    });

    expect(applied).toBe(false);
    expect(onUpdate).not.toHaveBeenCalled();
    unmount();
  });

  test("does not apply a parsed CEL response without an expression case", async () => {
    const onUpdate = vi.fn();
    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={{ type: ExprType.ConditionGroup, operator: "_&&_", args: [] }}
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        onUpdate={onUpdate}
      />
    );
    const askAgent = container.querySelector<HTMLButtonElement>(
      'button[aria-label="Ask Agent to help build this expression"]'
    );
    expressionMock.batchConvertCELStringToParsedExpr.mockResolvedValueOnce([
      { exprKind: {} as never },
    ]);

    let applied = true;
    await act(async () => {
      fireEvent.click(askAgent!);
      applied = await applyAgentExpressionDraft(
        "expression-chat",
        "invalid cel",
        "raw"
      );
    });

    expect(applied).toBe(false);
    expect(onUpdate).not.toHaveBeenCalled();
    unmount();
  });

  test("keeps Agent help beside the expression's Where label", () => {
    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={{
          type: ExprType.ConditionGroup,
          operator: "_&&_",
          args: [{ type: ExprType.RawString, content: "true" }],
        }}
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        onUpdate={() => {}}
      />
    );

    const askAgent = container.querySelector<HTMLButtonElement>(
      'button[aria-label="Ask Agent to help build this expression"]'
    );
    const whereLabel = Array.from(container.querySelectorAll("span")).find(
      (span) => span.textContent === "Where"
    );
    const expressionInput = container.querySelector("textarea");
    const whereLabelParent = whereLabel?.parentElement;
    if (!whereLabelParent || !expressionInput) {
      throw new Error("Expected the expression label and input to render");
    }
    expect(whereLabelParent.contains(askAgent)).toBe(true);
    expect(whereLabelParent.querySelector("textarea")).toBeNull();
    expect(
      whereLabelParent.compareDocumentPosition(expressionInput) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(askAgent?.className).toContain("text-accent");
    expect(askAgent?.className).not.toContain("border-control-border");
    unmount();
  });

  test("does not offer Agent help from a read-only expression", () => {
    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={{ type: ExprType.ConditionGroup, operator: "_&&_", args: [] }}
        readonly
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        onUpdate={() => {}}
      />
    );

    expect(
      container.querySelector(
        'button[aria-label="Ask Agent to help build this expression"]'
      )
    ).toBeNull();
    unmount();
  });

  test("revokes Agent handoffs when the source editor becomes read-only", async () => {
    const onUpdate = vi.fn();
    const renderEditor = (readonly: boolean) => (
      <ExprEditor
        expr={{ type: ExprType.ConditionGroup, operator: "_&&_", args: [] }}
        readonly={readonly}
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        onUpdate={onUpdate}
      />
    );
    const { container, rerender, unmount } = renderIntoContainer(
      renderEditor(false)
    );
    const askAgent = container.querySelector<HTMLButtonElement>(
      'button[aria-label="Ask Agent to help build this expression"]'
    );
    act(() => {
      fireEvent.click(askAgent!);
    });
    expect(canApplyAgentExpressionDraft("expression-chat", "raw")).toBe(true);

    rerender(renderEditor(true));

    expect(canApplyAgentExpressionDraft("expression-chat", "raw")).toBe(false);
    expect(canApplyAgentExpressionDraft("expression-chat", "selector")).toBe(
      false
    );
    await expect(
      applyAgentExpressionDraft("expression-chat", "true", "raw")
    ).resolves.toBe(false);
    expect(onUpdate).not.toHaveBeenCalled();
    unmount();
  });

  test("does not apply raw CEL after the editor becomes read-only during parsing", async () => {
    const onUpdate = vi.fn();
    const renderEditor = (readonly: boolean) => (
      <ExprEditor
        expr={{ type: ExprType.ConditionGroup, operator: "_&&_", args: [] }}
        readonly={readonly}
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        onUpdate={onUpdate}
      />
    );
    const { container, rerender, unmount } = renderIntoContainer(
      renderEditor(false)
    );
    let finishParsing!: (
      value: Awaited<
        ReturnType<typeof expressionMock.batchConvertCELStringToParsedExpr>
      >
    ) => void;
    expressionMock.batchConvertCELStringToParsedExpr.mockReturnValueOnce(
      new Promise((resolve) => {
        finishParsing = resolve;
      })
    );
    const askAgent = container.querySelector<HTMLButtonElement>(
      'button[aria-label="Ask Agent to help build this expression"]'
    );
    act(() => {
      fireEvent.click(askAgent!);
    });
    const applying = applyAgentExpressionDraft(
      "expression-chat",
      "true",
      "raw"
    );

    rerender(renderEditor(true));
    finishParsing([{ exprKind: { case: "callExpr", value: {} } }]);

    await expect(applying).resolves.toBe(false);
    expect(onUpdate).not.toHaveBeenCalled();
    unmount();
  });

  test("rejects an in-flight selector draft after ending and reopening Edit", async () => {
    const onUpdate = vi.fn();
    const renderEditor = (readonly: boolean) => (
      <ExprEditor
        expr={{ type: ExprType.ConditionGroup, operator: "_&&_", args: [] }}
        readonly={readonly}
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        onUpdate={onUpdate}
      />
    );
    const { container, rerender, unmount } = renderIntoContainer(
      renderEditor(false)
    );
    let finishValidation!: (value: string[]) => void;
    expressionMock.batchConvertParsedExprToCELString.mockReturnValueOnce(
      new Promise((resolve) => {
        finishValidation = resolve;
      })
    );
    const askAgent = container.querySelector<HTMLButtonElement>(
      'button[aria-label="Ask Agent to help build this expression"]'
    );
    act(() => {
      fireEvent.click(askAgent!);
    });
    const applying = applyAgentExpressionDraft(
      "expression-chat",
      "true",
      "selector"
    );
    await waitFor(() => {
      expect(
        expressionMock.batchConvertParsedExprToCELString
      ).toHaveBeenCalledOnce();
    });

    rerender(renderEditor(true));
    rerender(renderEditor(false));
    finishValidation(["canonical", "canonical"]);

    await expect(applying).resolves.toBe(false);
    expect(onUpdate).not.toHaveBeenCalled();
    expect(canApplyAgentExpressionDraft("expression-chat", "selector")).toBe(
      false
    );
    unmount();
  });

  test("applies a generated expression as raw CEL", async () => {
    const onUpdate = vi.fn();
    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={{ type: ExprType.ConditionGroup, operator: "_&&_", args: [] }}
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        onUpdate={onUpdate}
      />
    );
    const askAgent = container.querySelector<HTMLButtonElement>(
      'button[aria-label="Ask Agent to help build this expression"]'
    );
    await act(async () => {
      fireEvent.click(askAgent!);
      await applyAgentExpressionDraft(
        "expression-chat",
        'resource.environment == "prod"',
        "raw"
      );
    });
    expect(onUpdate).toHaveBeenCalledWith({
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.RawString,
          content: 'resource.environment == "prod"',
        },
      ],
    });
    unmount();
  });

  test.each(["_==_", "@in"] as const)(
    "cancels replaced searches and survives StrictMode replay for %s",
    async (operator) => {
      const signals: AbortSignal[] = [];
      const config: OptionConfig = {
        options: [],
        search: async ({ signal }) => {
          signals.push(signal!);
          return { options: [], nextPageToken: "" };
        },
      };
      const { container, unmount } = renderIntoContainer(
        <StrictMode>
          <ExprEditor
            expr={{
              type: ExprType.ConditionGroup,
              operator: "_&&_",
              args: [
                operator === "@in"
                  ? {
                      type: ExprType.Condition,
                      operator,
                      args: [CEL_ATTRIBUTE_RESOURCE_DATABASE, ["selected"]],
                    }
                  : {
                      type: ExprType.Condition,
                      operator,
                      args: [CEL_ATTRIBUTE_RESOURCE_DATABASE, "selected"],
                    },
              ],
            }}
            factorList={[CEL_ATTRIBUTE_RESOURCE_DATABASE]}
            optionConfigMap={
              new Map([[CEL_ATTRIBUTE_RESOURCE_DATABASE, config]])
            }
            onUpdate={() => {}}
          />
        </StrictMode>
      );
      await flushEffects();
      expect(signals).toHaveLength(2);
      expect(signals[0].aborted).toBe(true);
      expect(signals[1].aborted).toBe(false);
      const trigger =
        operator === "@in"
          ? Array.from(container.querySelectorAll("div")).find((el) =>
              el.className.includes("cursor-pointer")
            )
          : Array.from(container.querySelectorAll("button")).find(
              (el) => el.textContent === "selected"
            );
      expect(trigger).toBeTruthy();
      await act(async () => {
        fireEvent.click(trigger!);
      });
      expect(signals[1].aborted).toBe(true);
      const input = document.querySelector(
        'input[placeholder="common.filter-by-name"]'
      );
      expect(input).toBeTruthy();
      await act(async () => {
        fireEvent.change(input!, { target: { value: "new" } });
      });
      await waitFor(() => expect(signals[2].aborted).toBe(true));
      expect(signals[3].aborted).toBe(false);
      unmount();
      expect(signals[3].aborted).toBe(true);
    }
  );

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
      container.querySelectorAll("svg.lucide-trash-2")
    ).map((icon) => icon.closest("button"));
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

  test("hides expression editing actions in read-only mode", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.ConditionGroup,
          operator: "_&&_",
          args: [
            {
              type: ExprType.Condition,
              operator: "@in",
              args: [CEL_ATTRIBUTE_STATEMENT_SQL_TYPE, ["DDL"]],
            },
          ],
        },
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RISK_LEVEL, ["HIGH"]],
        },
      ],
    };

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        readonly
        factorList={[
          CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
          CEL_ATTRIBUTE_RISK_LEVEL,
        ]}
        optionConfigMap={optionConfigMap}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    expect(container.textContent).toContain("Where");
    expect(container.textContent).toContain("DDL");
    expect(container.textContent).not.toContain("cel.condition.add");
    expect(container.textContent).not.toContain(
      "cel.condition.add-raw-expression"
    );

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

  test("tag input uses only the outer field border", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, []],
        },
      ],
    };

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_TABLE_NAME]}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const input = container.querySelector(
      'input[placeholder="cel.condition.input-value-press-enter"]'
    );

    expect(input).toBeInstanceOf(HTMLInputElement);
    expect(input?.className).toContain("border-0");
    expect(input?.className).toContain("focus:ring-0");
    expect(input?.className).toContain("focus:border-0");
    expect(input?.className).toContain("placeholder:text-control-placeholder");

    unmount();
  });

  test("plain value input aligns with select trigger sizing", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "_==_",
          args: [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, ""],
        },
      ],
    };

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_TABLE_NAME]}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const input = container.querySelector(
      'input[placeholder="cel.condition.input-value"]'
    );

    expect(input).toBeInstanceOf(HTMLInputElement);
    expectClasses(
      input?.className,
      stylex.props(controlSizeStyle("md")).className ?? ""
    );
    expect(input?.className).not.toContain(
      stylex.props(controlSizeStyle("sm")).className ?? ""
    );

    const valueField = input?.parentElement;
    expect(valueField?.className).toContain("w-56");
    expect(valueField?.className).toContain("shrink-0");
    expect(input?.className).toContain("w-full");
    const deleteButton = container
      .querySelector("svg.lucide-trash-2")
      ?.closest("button");
    expect(deleteButton?.previousElementSibling).toBe(valueField);

    unmount();
  });

  test("multi value input aligns with select trigger sizing", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, []],
        },
      ],
    };

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_TABLE_NAME]}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const input = container.querySelector(
      'input[placeholder="cel.condition.input-value-press-enter"]'
    );
    const field = input?.parentElement;

    expect(field).toBeInstanceOf(HTMLDivElement);
    expect(field?.className).toContain("min-h-9");
    expect(field?.className).not.toContain("min-h-8");
    expect(field?.className).toContain("w-full");
    expect(field?.className).toContain("min-w-0");
    expect(field?.className).not.toContain("min-w-64");

    unmount();
  });

  test("static multi-select value aligns with select trigger sizing", async () => {
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

    const trigger = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("cel.condition.select-value")
    );

    expect(trigger).toBeInstanceOf(HTMLButtonElement);
    expect(trigger?.className).toContain("min-h-9");
    expect(trigger?.className).not.toContain("min-h-8");

    unmount();
  });

  test("searchable multi-select value aligns with select trigger sizing", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RESOURCE_DATABASE, []],
        },
      ],
    };

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_DATABASE]}
        optionConfigMap={searchableOptionConfigMap}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const placeholder = Array.from(container.querySelectorAll("span")).find(
      (span) => span.textContent?.includes("cel.condition.select-value")
    );
    const trigger = placeholder?.parentElement;

    expect(trigger).toBeInstanceOf(HTMLDivElement);
    expect(trigger?.className).toContain("min-h-9");
    expect(trigger?.className).not.toContain("min-h-8");

    unmount();
  });

  test("searchable multi-select dropdown can exceed its trigger width", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RESOURCE_DATABASE, []],
        },
      ],
    };

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_DATABASE]}
        optionConfigMap={searchableOptionConfigMap}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const trigger = Array.from(container.querySelectorAll("div")).find(
      (element) => element.className.includes("cursor-pointer")
    );
    expect(trigger).toBeInstanceOf(HTMLDivElement);

    await act(async () => {
      trigger?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      await Promise.resolve();
    });

    const filterInput = Array.from(document.body.querySelectorAll("input")).find(
      (input) => input.placeholder === "common.filter-by-name"
    );
    const dropdown = filterInput?.parentElement?.parentElement?.parentElement;

    expect(dropdown).toBeInstanceOf(HTMLDivElement);
    expect(dropdown?.className).toContain("w-80");
    expect(dropdown?.getAttribute("style")).not.toContain("width");

    unmount();
  });

  test("collection operator uses tag input when option config disables multi-select", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, []],
        },
      ],
    };
    const optionConfig = searchableOptionConfigMap.get(
      CEL_ATTRIBUTE_RESOURCE_TABLE_NAME
    ) as OptionConfig;

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_TABLE_NAME]}
        optionConfigMap={
          new Map([
            [
              CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
              {
                ...optionConfig,
                supportMultiple: false,
              },
            ],
          ])
        }
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    expect(
      container.querySelector(
        'input[placeholder="cel.condition.input-value-press-enter"]'
      )
    ).toBeInstanceOf(HTMLInputElement);
    expect(container.textContent?.includes("cel.condition.select-value")).toBe(
      false
    );

    unmount();
  });

  test("constrains long selected CEL values to one line", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RESOURCE_DATABASE, ["long-value"]],
        },
      ],
    };
    const longLabel = "prod instance (prod-sample-instance)";
    const selectedValueOptions = new Map<Factor, OptionConfig>([
      [
        CEL_ATTRIBUTE_RESOURCE_DATABASE,
        {
          options: [],
          fetch: async () => [{ value: "long-value", label: longLabel }],
          search: async () => ({ options: [], nextPageToken: "" }),
        },
      ],
    ]);

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_DATABASE]}
        optionConfigMap={selectedValueOptions}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const tag = Array.from(container.querySelectorAll("span")).find(
      (span) =>
        span.textContent === longLabel && span.className.includes("max-w-full")
    );

    expect(tag).toBeInstanceOf(HTMLSpanElement);
    expect(tag?.querySelector("span")?.className).toContain("truncate");

    unmount();
  });

  test("left-aligns selected environment values", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID, ["test"]],
        },
      ],
    };
    const environmentOptionConfig = new Map<Factor, OptionConfig>([
      [
        CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
        { options: [{ value: "test", label: "Test" }] },
      ],
    ]);

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        optionConfigMap={environmentOptionConfig}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const trigger = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("environments/test")
    );

    expect(trigger).toBeInstanceOf(HTMLButtonElement);
    expect(trigger?.className).toContain("justify-start");

    unmount();
  });

  test("searchable value trigger aligns with select trigger sizing", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "_==_",
          args: [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, ""],
        },
      ],
    };

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_TABLE_NAME]}
        optionConfigMap={searchableOptionConfigMap}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const trigger = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("cel.condition.select-value")
    );

    expect(trigger).toBeInstanceOf(HTMLButtonElement);
    expect(trigger?.className).toContain("h-9");
    expect(trigger?.className).not.toContain("h-8");
    expect(trigger?.querySelector("span")?.className).toContain("flex-1");
    expect(trigger?.querySelector("span")?.className).toContain("truncate");

    unmount();
  });

  test("shows the select placeholder for an empty static option value", async () => {
    const initialExpr: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "_==_",
          args: [CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID, ""],
        },
      ],
    };
    const environmentOptionConfig = new Map<Factor, OptionConfig>([
      [
        CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
        { options: [{ value: "test", label: "Test" }] },
      ],
    ]);

    const { container, unmount } = renderIntoContainer(
      <ExprEditor
        expr={initialExpr}
        factorList={[CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]}
        optionConfigMap={environmentOptionConfig}
        onUpdate={() => {}}
      />
    );
    await flushEffects();

    const placeholder = Array.from(container.querySelectorAll("span")).find(
      (span) =>
        span.textContent === "cel.condition.select-value" &&
        span.className.includes("text-control-placeholder")
    );

    expect(placeholder).toBeInstanceOf(HTMLSpanElement);

    unmount();
  });
});
