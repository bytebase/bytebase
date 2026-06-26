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

const mocks = vi.hoisted(() => ({
  useUnsavedChangesGuard: vi.fn(),
  upsertReviewPolicy: vi.fn(),
  getOrFetchReviewPolicyByName: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => true),
  inputOnChange: vi.fn(),
  resourceIdOnChange: vi.fn(),
  ruleTableProps: [] as unknown[],
}));

let ReviewCreation: typeof import("./ReviewCreation").ReviewCreation;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: mocks.useUnsavedChangesGuard,
}));

vi.mock("@/react/router", () => ({
  router: {
    push: vi.fn(),
  },
}));

vi.mock("@/react/stores/sqlReview", () => ({
  useSQLReviewStore: () => ({
    upsertReviewPolicy: mocks.upsertReviewPolicy,
    getOrFetchReviewPolicyByName: mocks.getOrFetchReviewPolicyByName,
  }),
}));

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
  sqlReviewPolicySlug: () => "review",
}));

vi.mock("@/types", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/types")>();
  return {
    ...actual,
    TEMPLATE_LIST_V2: [
      {
        id: "bb.sql-review.empty",
        ruleList: [
          {
            type: 110,
            category: "BUILTIN",
            engine: 2,
            level: 1,
            componentList: [],
          },
        ],
      },
    ],
  };
});

vi.mock("@/react/components/ResourceIdField", () => ({
  ResourceIdField: ({
    value,
    onChange,
  }: {
    value: string;
    onChange: (value: string) => void;
  }) => {
    mocks.resourceIdOnChange = vi.fn(onChange);
    return (
      <input
        aria-label="resource-id"
        value={value}
        onChange={(event) => onChange(event.currentTarget.value)}
      />
    );
  },
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

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => {
    mocks.inputOnChange = vi.fn(props.onChange);
    return <input {...props} />;
  },
}));

vi.mock("./TemplateSelector", () => ({
  TemplateSelector: () => <div data-testid="template-selector" />,
}));

vi.mock("./TabsByEngine", () => ({
  TabsByEngine: ({
    ruleMapByEngine,
    children,
  }: {
    ruleMapByEngine: Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>;
    children: (ruleList: RuleTemplateV2[], engine: Engine) => React.ReactNode;
  }) => (
    <div data-testid="tabs-by-engine">
      {[...ruleMapByEngine.entries()].map(([engine, ruleMap]) => (
        <div key={engine}>{children([...ruleMap.values()], engine)}</div>
      ))}
    </div>
  ),
}));

vi.mock("./RuleTable", () => ({
  RuleTableWithFilter: (props: unknown) => {
    mocks.ruleTableProps.push(props);
    return <div data-testid="rule-table" />;
  },
}));

vi.mock("./Panels", () => ({
  RulesSelectPanel: () => <div data-testid="rules-select-panel" />,
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

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.ruleTableProps = [];
  ({ ReviewCreation } = await import("./ReviewCreation"));
});

const invalidStringArrayRule: RuleTemplateV2 = {
  type: SQLReviewRule_Type.TABLE_DISALLOW_DDL,
  category: "TABLE",
  engine: Engine.MYSQL,
  level: SQLReviewRule_Level.WARNING,
  componentList: [
    {
      key: "list",
      payload: {
        type: "STRING_ARRAY",
        default: [],
        value: [],
      },
    },
  ],
};

describe("ReviewCreation", () => {
  test("starts from scratch with built-in rules in create mode", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ReviewCreation selectedRuleList={[]} selectedResources={[]} />
    );

    render();

    act(() => {
      mocks.resourceIdOnChange("custom-policy");
    });
    const nextButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.next"
    );
    expect(nextButton).toBeTruthy();

    act(() => {
      nextButton!.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(mocks.ruleTableProps).toHaveLength(1);
    const ruleList = (
      mocks.ruleTableProps[0] as { ruleList?: RuleTemplateV2[] }
    ).ruleList;
    expect(ruleList?.length).toBeGreaterThan(0);
    expect(ruleList?.every((rule) => rule.category === "BUILTIN")).toBe(true);

    unmount();
  });

  test("guards navigation only after SQL review form changes", () => {
    const { render, unmount } = renderIntoContainer(
      <ReviewCreation selectedRuleList={[]} selectedResources={[]} />
    );

    render();

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    act(() => {
      mocks.inputOnChange({
        target: { value: "Custom policy" },
      });
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);

    unmount();
  });

  test("places rule selection beside the finish action in the sticky footer", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ReviewCreation selectedRuleList={[]} selectedResources={[]} />
    );

    render();

    act(() => {
      mocks.resourceIdOnChange("custom-policy");
    });
    const nextButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.next"
    );
    expect(nextButton).toBeTruthy();

    act(() => {
      nextButton!.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const stickyFooter = container.querySelector(".sticky.bottom-0");
    expect(stickyFooter?.textContent).toContain(
      "sql-review.add-or-remove-rules"
    );
    expect(stickyFooter?.textContent).toContain("common.create");
    expect(
      stickyFooter?.textContent?.indexOf("sql-review.add-or-remove-rules")
    ).toBeLessThan(stickyFooter?.textContent?.indexOf("common.create") ?? -1);

    unmount();
  });

  test("focuses the invalid rule after validation blocks submit", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ReviewCreation
        policy={{
          id: "reviewConfigs/test",
          enforce: true,
          name: "Test policy",
          resources: [],
          ruleList: [],
        }}
        name="Test policy"
        selectedRuleList={[invalidStringArrayRule]}
        selectedResources={[]}
      />
    );

    render();

    const nextButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.next"
    );
    expect(nextButton).toBeTruthy();

    act(() => {
      nextButton!.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    const updateButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.update"
    );
    expect(updateButton).toBeTruthy();

    act(() => {
      updateButton!.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const lastRuleTableProps = mocks.ruleTableProps.at(-1) as {
      focusRuleKey?: string;
      focusRuleSignal?: number;
    };
    expect(lastRuleTableProps.focusRuleKey).toBe(
      `${Engine.MYSQL}:${SQLReviewRule_Type.TABLE_DISALLOW_DDL}`
    );
    expect(lastRuleTableProps.focusRuleSignal).toBeGreaterThan(0);

    unmount();
  });
});
