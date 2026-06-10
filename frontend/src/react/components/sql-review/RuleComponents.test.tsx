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

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
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
  ({ RuleEditDialog } = await import("./RuleComponents"));
});

describe("RuleEditDialog", () => {
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
