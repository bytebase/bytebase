import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { getRuleKey } from "@/lib/sql-review/utils";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";
import {
  getRuleLocalization,
  ruleTemplateMapV2,
  ruleTypeToString,
} from "@/types/sqlReview";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  ruleLevelSwitch: vi.fn(),
  searchInputOnChange: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/types/sqlReview", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/types/sqlReview")>();
  return {
    ...actual,
    getRuleLocalization: vi.fn(actual.getRuleLocalization),
  };
});

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    ...props
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  } & React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button onClick={onClick} {...props}>
      {children}
    </button>
  ),
}));

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: () => <input type="checkbox" readOnly />,
}));

vi.mock("@/components/ui/table", () => ({
  Table: ({ children }: { children: React.ReactNode }) => (
    <table>{children}</table>
  ),
  TableBody: ({ children }: { children: React.ReactNode }) => (
    <tbody>{children}</tbody>
  ),
  TableCell: ({
    children,
    ...props
  }: React.TdHTMLAttributes<HTMLTableCellElement>) => (
    <td {...props}>{children}</td>
  ),
  TableHead: ({
    children,
    ...props
  }: React.ThHTMLAttributes<HTMLTableCellElement>) => (
    <th {...props}>{children}</th>
  ),
  TableHeader: ({ children }: { children: React.ReactNode }) => (
    <thead>{children}</thead>
  ),
  TableRow: ({
    children,
    ...props
  }: React.HTMLAttributes<HTMLTableRowElement>) => (
    <tr {...props}>{children}</tr>
  ),
}));

vi.mock("@/components/ui/search-input", () => ({
  SearchInput: (props: React.InputHTMLAttributes<HTMLInputElement>) => {
    mocks.searchInputOnChange = vi.fn(props.onChange);
    return <input aria-label="rule-search" {...props} />;
  },
}));

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({
    children,
    onValueChange: _onValueChange,
    value: _value,
  }: {
    children: React.ReactNode;
    onValueChange?: (value: string) => void;
    value?: string;
  }) => <div>{children}</div>,
  TabsList: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  TabsTrigger: ({
    children,
    value,
    onClick,
  }: {
    children: React.ReactNode;
    value: string;
    onClick?: () => void;
  }) => (
    <button data-value={value} onClick={onClick}>
      {children}
    </button>
  ),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock("./RuleComponents", () => ({
  RuleConfig: () => <div data-testid="rule-config" />,
  RuleEditDialog: () => <div data-testid="rule-edit-dialog" />,
  RuleLevelFilter: () => <div data-testid="rule-level-filter" />,
  RuleLevelSwitch: () => {
    mocks.ruleLevelSwitch();
    return <button>level</button>;
  },
}));

let RuleTable: typeof import("./RuleTable").RuleTable;
let RuleTableWithFilter: typeof import("./RuleTable").RuleTableWithFilter;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  return {
    container,
    render: (nextElement = element) =>
      act(() => {
        root.render(nextElement);
      }),
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  Object.defineProperty(window, "requestAnimationFrame", {
    configurable: true,
    value: (callback: FrameRequestCallback) => {
      callback(0);
      return 0;
    },
  });
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  });
  Object.defineProperty(Element.prototype, "scrollIntoView", {
    configurable: true,
    value: vi.fn(),
  });
  ({ RuleTable, RuleTableWithFilter } = await import("./RuleTable"));
});

describe("RuleTable", () => {
  const ruleList = [
    {
      value: "all",
      label: "All",
      ruleList: [
        ...(ruleTemplateMapV2.get(Engine.MYSQL)?.values() ?? []),
      ].slice(0, 20) as RuleTemplateV2[],
    },
  ];
  const getRuleWithPayload = () => {
    const rule = ruleList[0].ruleList.find(
      (rule) => rule.componentList.length > 0
    );
    if (!rule) {
      throw new Error("expected a SQL review rule with payload");
    }
    return rule;
  };
  const getRuleWithoutPayload = () => {
    const rule = ruleList[0].ruleList.find((rule) => {
      if (rule.componentList.length > 0) {
        return false;
      }
      return !!getRuleLocalization(
        ruleTypeToString(rule.type),
        rule.engine
      ).description;
    });
    if (!rule) {
      throw new Error("expected a SQL review rule without payload");
    }
    return rule;
  };

  test("opening one edit dialog does not rerender every visible rule row", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleTable ruleList={ruleList} editable />
    );

    render();
    const callCountAfterInitialRender =
      vi.mocked(getRuleLocalization).mock.calls.length;
    const levelSwitchRenderCountAfterInitialRender =
      mocks.ruleLevelSwitch.mock.calls.length;

    const editButton = container.querySelector<HTMLButtonElement>(
      'button[aria-label="common.edit"]'
    );
    expect(editButton).toBeTruthy();
    expect(editButton?.textContent).toBe("");

    act(() => {
      editButton!.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(
      container.querySelector('[data-testid="rule-edit-dialog"]')
    ).toBeTruthy();
    expect(vi.mocked(getRuleLocalization).mock.calls.length).toBeLessThan(
      callCountAfterInitialRender + 5
    );
    expect(mocks.ruleLevelSwitch.mock.calls.length).toBeLessThan(
      levelSwitchRenderCountAfterInitialRender + 5
    );

    unmount();
  });

  test("desktop rule row uses compact edit action and aligned expanded details", () => {
    const rule = getRuleWithPayload();
    const { container, render, unmount } = renderIntoContainer(
      <RuleTable
        ruleList={[{ value: "all", label: "All", ruleList: [rule] }]}
        editable
      />
    );

    render();

    const editButton = container.querySelector<HTMLButtonElement>(
      'tbody tr[data-sql-review-rule-view="desktop"] button[aria-label="common.edit"]'
    );
    expect(editButton).toBeTruthy();
    expect(editButton?.className).toContain("size-7");
    expect(editButton?.textContent).toBe("");
    expect(editButton?.className).not.toContain("border");

    const deleteButton = container.querySelector<HTMLButtonElement>(
      'tbody tr[data-sql-review-rule-view="desktop"] button[aria-label="common.delete"]'
    );
    expect(deleteButton).toBeTruthy();
    expect(deleteButton?.className).toContain("size-7");
    expect(deleteButton?.textContent).toBe("");
    expect(deleteButton?.className).not.toContain("border");
    expect(editButton?.parentElement?.className).toContain("gap-x-1");
    const expandCell = container.querySelector(
      'tbody tr[data-sql-review-rule-view="desktop"] td:first-child'
    );
    expect(expandCell?.className).toContain("align-top");
    expect(expandCell?.className).toContain("pt-4");

    const levelCell = container.querySelector(
      'tbody tr[data-sql-review-rule-view="desktop"] td:nth-child(3)'
    );
    expect(levelCell?.className).toContain("align-top");
    expect(levelCell?.className).toContain("pt-4");

    const operationsCell = container.querySelector(
      'tbody tr[data-sql-review-rule-view="desktop"] td:nth-child(4)'
    );
    expect(operationsCell?.className).toContain("align-top");
    expect(operationsCell?.className).toContain("pt-4");

    const title = container.querySelector(
      'tbody tr[data-sql-review-rule-view="desktop"] td:nth-child(2) span'
    );
    expect(title?.className).toContain("font-semibold");
    expect(title?.className).toContain("text-base");
    expect(title?.className).toContain("text-main");

    const description = container.querySelector(
      'tbody tr[data-sql-review-rule-view="desktop"] td:nth-child(2) p'
    );
    expect(description).toBeTruthy();
    expect(description?.className).toContain("text-sm");
    expect(description?.className).toContain("leading-5");
    expect(description?.className).toContain("text-control-light");

    const expandButton = container.querySelector<HTMLButtonElement>(
      "tbody tr[data-sql-review-rule-view='desktop'] td:first-child button"
    );
    expect(expandButton).toBeTruthy();
    act(() => {
      expandButton!.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const rows = container.querySelectorAll("tbody tr");
    const detailRow = rows[1];
    expect(detailRow?.className).toContain("bg-control-bg/20");

    const detailCells = detailRow?.querySelectorAll("td");
    expect(detailCells).toHaveLength(2);
    expect(detailCells?.[0]?.textContent).toBe("");
    expect(detailCells?.[1]?.getAttribute("colspan")).toBe("3");
    expect(detailCells?.[1]?.className).toContain("px-4");
    expect(detailCells?.[1]?.className).not.toContain("px-10");
    expect(
      detailCells?.[1]?.querySelector('[data-testid="rule-config"]')
    ).toBeTruthy();
    expect(detailCells?.[1]?.querySelector("p")).toBeFalsy();

    unmount();
  });

  test("desktop rule row only expands rules with payload", () => {
    const rule = getRuleWithoutPayload();
    const { container, render, unmount } = renderIntoContainer(
      <RuleTable
        ruleList={[{ value: "all", label: "All", ruleList: [rule] }]}
        editable
      />
    );

    render();

    expect(
      container.querySelector(
        'tbody tr[data-sql-review-rule-view="desktop"] td:nth-child(2) p'
      )
    ).toBeTruthy();
    expect(
      container.querySelector(
        "tbody tr[data-sql-review-rule-view='desktop'] td:first-child button"
      )
    ).toBeFalsy();

    unmount();
  });

  test("expanding one rule does not rerender every visible rule row", () => {
    const rule = getRuleWithPayload();
    const { container, render, unmount } = renderIntoContainer(
      <RuleTable
        ruleList={[{ value: "all", label: "All", ruleList: [rule] }]}
        editable
      />
    );

    render();
    const levelSwitchRenderCountAfterInitialRender =
      mocks.ruleLevelSwitch.mock.calls.length;
    const rowCountAfterInitialRender =
      container.querySelectorAll("tbody tr").length;

    const expandButton = container.querySelector("tbody button");
    expect(expandButton).toBeTruthy();

    act(() => {
      expandButton!.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(container.querySelectorAll("tbody tr").length).toBeGreaterThan(
      rowCountAfterInitialRender
    );
    expect(mocks.ruleLevelSwitch.mock.calls.length).toBeLessThan(
      levelSwitchRenderCountAfterInitialRender + 5
    );

    unmount();
  });

  test("repeated focus signals clear active filters for the same invalid rule", () => {
    const rule = ruleList[0].ruleList[0];
    const focusRuleKey = getRuleKey(rule);
    const { container, render, unmount } = renderIntoContainer(
      <RuleTableWithFilter
        engine={Engine.MYSQL}
        ruleList={[rule]}
        editable
        focusRuleKey={focusRuleKey}
        focusRuleSignal={1}
      />
    );

    render();

    const searchInput = container.querySelector<HTMLInputElement>(
      'input[aria-label="rule-search"]'
    );
    expect(searchInput).toBeTruthy();
    act(() => {
      mocks.searchInputOnChange({
        target: { value: "does-not-match-any-rule" },
      });
    });
    expect(container.textContent).toContain("common.no-data");

    render(
      <RuleTableWithFilter
        engine={Engine.MYSQL}
        ruleList={[rule]}
        editable
        focusRuleKey={focusRuleKey}
        focusRuleSignal={2}
      />
    );

    expect(container.textContent).not.toContain("common.no-data");

    unmount();
  });

  test("focus scroll targets the mobile rule card on small screens", () => {
    vi.mocked(window.matchMedia).mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    } as unknown as MediaQueryList);
    const scrollTargets: Element[] = [];
    Object.defineProperty(Element.prototype, "scrollIntoView", {
      configurable: true,
      value: vi.fn(function (this: Element) {
        scrollTargets.push(this);
      }),
    });
    const rule = ruleList[0].ruleList[0];
    const focusRuleKey = getRuleKey(rule);
    const { render, unmount } = renderIntoContainer(
      <RuleTable
        ruleList={[{ value: "all", label: "All", ruleList: [rule] }]}
        editable
        focusRuleKey={focusRuleKey}
        focusRuleSignal={1}
      />
    );

    render();

    expect(scrollTargets[0]?.tagName).toBe("DIV");

    unmount();
  });

  test("mobile rule row uses compact matched actions with vertical padding", () => {
    const rule = {
      ...ruleList[0].ruleList[0],
      category: "ENGINE",
    };
    const { container, render, unmount } = renderIntoContainer(
      <RuleTable
        ruleList={[{ value: "all", label: "All", ruleList: [rule] }]}
        editable
      />
    );

    render();

    const mobileRow = container.querySelector(
      '[data-sql-review-rule-view="mobile"]'
    );
    expect(mobileRow).toBeTruthy();
    expect(mobileRow?.className).toContain("py-4");
    expect(mobileRow?.className).not.toContain("pt-4");
    expect(
      mobileRow?.querySelector('[data-testid="mobile-rule-title-row"]')
        ?.className
    ).not.toContain("grid-cols-[minmax(0,1fr)_auto]");
    expect(
      mobileRow?.querySelector('[data-testid="mobile-rule-action-list"]')
        ?.className
    ).toContain("absolute");
    const mobileEditButton = mobileRow?.querySelector<HTMLButtonElement>(
      'button[aria-label="common.edit"]'
    );
    const mobileDeleteButton = mobileRow?.querySelector<HTMLButtonElement>(
      'button[aria-label="common.delete"]'
    );
    expect(mobileEditButton).toBeTruthy();
    expect(mobileDeleteButton).toBeTruthy();
    expect(mobileEditButton?.className).toContain("size-7");
    expect(mobileDeleteButton?.className).toContain("size-7");
    expect(mobileEditButton?.className).not.toContain("size-8");
    expect(mobileDeleteButton?.className).not.toContain("size-8");
    expect(mobileDeleteButton?.textContent).toBe("");

    unmount();
  });
});
