import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  getOrFetchReviewPolicyByName: vi.fn(),
  getReviewPolicyByName: vi.fn(),
  removeReviewPolicy: vi.fn(),
  replace: vi.fn(),
  upsertReviewPolicy: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({
  router: {
    replace: mocks.replace,
  },
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/components/sql-review/Panels", () => ({
  AttachResourcesPanel: () => null,
}));

vi.mock("@/components/sql-review/ResourceLink", () => ({
  ResourceLink: ({ resource }: { resource: string }) => <span>{resource}</span>,
}));

vi.mock("@/components/sql-review/RuleTable", () => ({
  RuleTableWithFilter: ({
    onRuleUpsert,
  }: {
    onRuleUpsert: (rule: { engine: number; type: number }, update: object) => void;
  }) => (
    <button
      type="button"
      onClick={() => onRuleUpsert({ engine: 1, type: 1 }, { level: 2 })}
    >
      Change rule
    </button>
  ),
}));

vi.mock("@/components/sql-review/TabsByEngine", () => ({
  TabsByEngine: ({
    children,
  }: {
    children: (ruleList: unknown[], engine: number) => ReactNode;
  }) => <div>{children([], 1)}</div>,
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/stores/sqlReview", () => ({
  useSQLReviewStore: Object.assign(
    () => ({
      getReviewPolicyByName: mocks.getReviewPolicyByName,
      removeReviewPolicy: mocks.removeReviewPolicy,
      upsertReviewPolicy: mocks.upsertReviewPolicy,
    }),
    {
      getState: () => ({
        getOrFetchReviewPolicyByName: mocks.getOrFetchReviewPolicyByName,
      }),
    }
  ),
}));

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: vi.fn(() => true),
  setDocumentTitle: vi.fn(),
  sqlReviewNameFromSlug: (slug: string) => slug,
}));

let SQLReviewDetailPage: typeof import("./SQLReviewDetailPage").SQLReviewDetailPage;

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

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.getReviewPolicyByName.mockReturnValue({
    id: "policy-1",
    enforce: true,
    name: "Sample SQL Review Config",
    ruleList: [],
    resources: ["projects/sample"],
  });
  ({ SQLReviewDetailPage } = await import("./SQLReviewDetailPage"));
});

describe("SQLReviewDetailPage", () => {
  test("keeps changed-rule actions at the bottom without extra side inset", () => {
    const { container, unmount } = renderIntoContainer(
      <SQLReviewDetailPage sqlReviewPolicySlug="policy-1" />
    );

    expect(container.firstElementChild).toHaveClass(
      "flex",
      "h-full",
      "min-h-0",
      "flex-col",
      "overflow-hidden"
    );

    act(() => {
      [...container.querySelectorAll("button")]
        .find((button) => button.textContent === "Change rule")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const footer = container.querySelector('[data-slot="sticky-action-footer"]');
    const scrollContent = container.querySelector(
      '[data-slot="sql-review-detail-content"]'
    );
    const content = container.querySelector(
      '[data-slot="sticky-action-footer-content"]'
    );

    expect(scrollContent).toHaveClass("min-h-0", "flex-1", "overflow-y-auto");
    expect(footer?.previousElementSibling).toBe(scrollContent);
    expect(footer).toHaveClass("shrink-0");
    expect(footer).not.toHaveClass("mt-4");
    expect(footer).not.toHaveClass("mt-auto");
    expect(content).toHaveClass("px-4", "sm:px-6", "gap-x-2", "gap-y-2");

    unmount();
  });

  test("keeps the delete policy action at its natural button width", () => {
    const { container, unmount } = renderIntoContainer(
      <SQLReviewDetailPage sqlReviewPolicySlug="policy-1" />
    );

    const deleteButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "sql-review.delete"
    );

    expect(deleteButton).toHaveClass("self-start");
    expect(deleteButton).not.toHaveClass("w-full");

    unmount();
  });
});
