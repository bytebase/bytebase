import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  SQLReviewRule_Level,
  SQLReviewRule_Type,
} from "@/types/proto-es/v1/review_config_service_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let RuleEditDialog: typeof import("./RuleComponents").RuleEditDialog;
let RuleLevelSwitch: typeof import("./RuleComponents").RuleLevelSwitch;
let RuleConfig: typeof import("./RuleComponents").RuleConfig;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/components/EngineIcon", () => ({
  EngineIcon: ({ engine }: { engine: Engine }) => (
    <span data-engine={engine} data-testid="engine-icon" />
  ),
}));

vi.mock("@/react/i18n", () => ({
  default: {
    exists: vi.fn(() => false),
    t: vi.fn((key: string) => key),
  },
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
  }: {
    children: React.ReactNode;
    disabled?: boolean;
    onClick?: () => void;
  }) => (
    <button disabled={disabled} onClick={onClick}>
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/checkbox", () => ({
  Checkbox: () => <input type="checkbox" readOnly />,
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h2>{children}</h2>
  ),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
}));

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

const tableDenyListRule = (list: string[]): RuleTemplateV2 => ({
  type: SQLReviewRule_Type.TABLE_DISALLOW_DDL,
  category: "TABLE",
  engine: Engine.MYSQL,
  level: SQLReviewRule_Level.ERROR,
  componentList: [
    {
      key: "list",
      payload: {
        type: "STRING_ARRAY",
        default: [],
        value: list,
      },
    },
  ],
});

beforeEach(async () => {
  ({ RuleConfig, RuleEditDialog, RuleLevelSwitch } = await import(
    "./RuleComponents"
  ));
});

describe("RuleLevelSwitch", () => {
  test("raises the active error segment above the neighboring warning segment", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleLevelSwitch
        level={SQLReviewRule_Level.ERROR}
        onLevelChange={vi.fn()}
      />
    );

    render();

    const errorButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "sql-review.level.error"
    );
    expect(errorButton).toBeTruthy();
    expect(errorButton?.className).toContain("z-10");

    unmount();
  });
});

describe("RuleEditDialog", () => {
  test("uses the rule title as the modal title", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleEditDialog
        rule={tableDenyListRule(["id"])}
        disabled={false}
        onUpdateRule={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    render();

    expect(container.querySelector("h2")?.textContent).toBe(
      "sql-review.rule.TABLE_DISALLOW_DDL.title"
    );

    unmount();
  });

  test("renders engine icon and name in the modal header", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleEditDialog
        rule={{ ...tableDenyListRule(["id"]), engine: Engine.POSTGRES }}
        disabled={false}
        onUpdateRule={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    render();

    expect(container.querySelector('[data-testid="engine-icon"]')).toBeTruthy();
    expect(container.textContent).toContain("PostgreSQL");

    unmount();
  });

  test("adds top spacing before the editable string-array input", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleConfig
        rule={tableDenyListRule(["id"])}
        disabled={false}
        size="medium"
      />
    );

    render();

    const input = container.querySelector("input");
    expect(input).toBeTruthy();
    expect(input?.className).toContain("mt-2");

    unmount();
  });

  test("renders read-only string-array labels with visible chip backgrounds", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleConfig rule={tableDenyListRule(["id"])} disabled size="small" />
    );

    render();

    const tag = [...container.querySelectorAll("span")].find(
      (span) => span.textContent === "id"
    );
    expect(tag).toBeTruthy();
    expect(tag?.className).toContain("bg-background");
    expect(tag?.className).toContain("border-control-border");

    unmount();
  });

  test("disables update for an empty required string-array payload", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleEditDialog
        rule={tableDenyListRule([])}
        disabled={false}
        onUpdateRule={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    render();

    const updateButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.update"
    );

    expect(updateButton).toBeTruthy();
    expect(updateButton).toBeDisabled();

    unmount();
  });

  test("enables update for a configured required string-array payload", () => {
    const { container, render, unmount } = renderIntoContainer(
      <RuleEditDialog
        rule={tableDenyListRule(["audit_log"])}
        disabled={false}
        onUpdateRule={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    render();

    const updateButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.update"
    );

    expect(updateButton).toBeTruthy();
    expect(updateButton).not.toBeDisabled();

    unmount();
  });
});
