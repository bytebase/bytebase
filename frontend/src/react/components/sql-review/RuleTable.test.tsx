import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { getRuleKey } from "@/react/lib/sql-review/utils";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";
import { getRuleLocalization, ruleTemplateMapV2 } from "@/types/sqlReview";

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

vi.mock("@/react/components/ui/button", () => ({
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

vi.mock("@/react/components/ui/checkbox", () => ({
  Checkbox: () => <input type="checkbox" readOnly />,
}));

vi.mock("@/react/components/ui/table", () => ({
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

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: (props: React.InputHTMLAttributes<HTMLInputElement>) => {
    mocks.searchInputOnChange = vi.fn(props.onChange);
    return <input aria-label="rule-search" {...props} />;
  },
}));

vi.mock("@/react/components/ui/tabs", () => ({
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

  test("opening one edit dialog does not rerender every visible rule row", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleTable ruleList={ruleList} editable />
    );

    render();
    const callCountAfterInitialRender =
      vi.mocked(getRuleLocalization).mock.calls.length;
    const levelSwitchRenderCountAfterInitialRender =
      mocks.ruleLevelSwitch.mock.calls.length;

    const editButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.edit"
    );
    expect(editButton).toBeTruthy();

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

  test("expanding one rule does not rerender every visible rule row", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleTable ruleList={ruleList} editable />
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
