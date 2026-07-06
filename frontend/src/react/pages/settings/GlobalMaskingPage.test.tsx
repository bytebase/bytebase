import type { InputHTMLAttributes, ReactNode } from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("@/plugins/cel", () => ({
  buildCELExpr: vi.fn(),
  emptySimpleExpr: () => ({}),
  resolveCELExpr: () => ({}),
  validateSimpleExpr: () => true,
  wrapAsGroup: (expr: unknown) => expr,
}));

vi.mock("@/react/components/ExprEditor", () => ({
  ExprEditor: () => createElement("div", { "data-testid": "expr-editor" }),
}));

vi.mock("@/react/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));

vi.mock("@/react/components/ui/alert", () => ({
  Alert: ({
    children,
    description,
  }: {
    children?: ReactNode;
    description?: ReactNode;
  }) => createElement("div", {}, description, children),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
  }: {
    children?: ReactNode;
    disabled?: boolean;
    onClick?: () => void;
  }) => createElement("button", { disabled, onClick }, children),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: InputHTMLAttributes<HTMLInputElement>) =>
    createElement("input", props),
}));

vi.mock("@/react/components/ui/select", () => ({
  Select: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SelectContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SelectItem: ({ children }: { children: ReactNode; value: string }) =>
    createElement("div", {}, children),
  SelectTrigger: ({
    children,
    className,
  }: {
    children?: ReactNode;
    className?: string;
  }) =>
    createElement(
      "button",
      { className, "data-testid": "semantic-type-trigger", type: "button" },
      children
    ),
  SelectValue: ({ children }: { children?: ReactNode }) =>
    createElement("span", {}, typeof children === "function" ? null : children),
}));

vi.mock("@/react/lib/sensitive-data/components-utils", () => ({
  factorOperatorOverrideMap: {},
  getClassificationLevelOptions: () => [],
}));

vi.mock("@/react/stores/app", () => {
  const state = {
    environmentList: [],
    getOrFetchPolicyByParentAndType: vi.fn(async () => ({
      policy: {
        case: "maskingRulePolicy",
        value: {
          rules: [
            {
              condition: { title: "Default masking" },
              id: "rule-1",
              semanticType: "DEFAULT",
            },
          ],
        },
      },
    })),
    getOrFetchSettingByName: vi.fn(async () => undefined),
    getSettingByName: () => ({
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [{ id: "DEFAULT", title: "Default" }],
          },
        },
      },
    }),
    hasInstanceFeature: () => true,
    settingsByName: {},
    workspaceResourceName: () => "workspaces/default",
  };
  type StoreState = typeof state;
  const useAppStore = (selector?: (store: StoreState) => unknown) =>
    selector ? selector(state) : state;
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/types/proto-es/google/type/expr_pb", () => ({
  ExprSchema: {},
}));

vi.mock("@/types/proto-es/v1/org_policy_service_pb", () => ({
  MaskingRulePolicy_MaskingRuleSchema: {},
  MaskingRulePolicySchema: {},
  PolicyResourceType: { WORKSPACE: 1 },
  PolicyType: { MASKING_RULE: 1 },
}));

vi.mock("@/types/proto-es/v1/setting_service_pb", () => ({
  Setting_SettingName: {
    DATA_CLASSIFICATION: 2,
    SEMANTIC_TYPES: 1,
  },
}));

vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: { FEATURE_DATA_MASKING: 1 },
}));

vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("lucide-react", () => ({
  ChevronDown: () => createElement("span", {}),
  ChevronUp: () => createElement("span", {}),
  ExternalLink: () => createElement("span", {}),
  ListOrdered: () => createElement("span", {}),
  Pencil: () => createElement("span", {}),
  Plus: () => createElement("span", {}),
  Trash2: () => createElement("span", {}),
}));

vi.mock("uuid", () => ({
  v4: () => "new-rule",
}));

vi.mock("@/utils", () => {
  return {
    arraySwap: (items: unknown[], a: number, b: number) => {
      [items[a], items[b]] = [items[b], items[a]];
    },
    batchConvertCELStringToParsedExpr: vi.fn(async () => []),
    batchConvertParsedExprToCELString: vi.fn(async () => [""]),
    getDatabaseIdOptionConfig: () => ({ options: [] }),
    getInstanceIdOptionConfig: () => ({ options: [] }),
    getProjectIdOptionConfig: () => ({ options: [] }),
    hasWorkspacePermissionV2: () => true,
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

import { GlobalMaskingPage } from "./GlobalMaskingPage";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.clearAllMocks();
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

async function renderPage(): Promise<void> {
  await act(async () => {
    root.render(createElement(GlobalMaskingPage));
    await Promise.resolve();
  });
}

describe("GlobalMaskingPage", () => {
  it("aligns the semantic type select with the expression editor", async () => {
    await renderPage();

    const semanticTypeHeading = Array.from(
      container.querySelectorAll("h3")
    ).find(
      (heading) =>
        heading.textContent ===
        "settings.sensitive-data.semantic-types.table.semantic-type"
    );

    expect(semanticTypeHeading?.className).toContain("h-9");
    expect(semanticTypeHeading?.className).toContain("items-center");
    expect(semanticTypeHeading?.parentElement?.className).toContain("gap-y-2");
    expect(
      container.querySelector("[data-testid='expr-editor']")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='semantic-type-trigger']")
    ).not.toBeNull();
  });
});
