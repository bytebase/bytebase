import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";
import { getRuleLocalization, ruleTemplateMapV2 } from "@/types/sqlReview";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  ruleLevelSwitch: vi.fn(),
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
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => <button onClick={onClick}>{children}</button>,
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

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () =>
      act(() => {
        root.render(element);
      }),
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  Object.defineProperty(Element.prototype, "scrollIntoView", {
    configurable: true,
    value: vi.fn(),
  });
  ({ RuleTable } = await import("./RuleTable"));
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
});
