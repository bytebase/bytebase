import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLReviewPolicyTemplateV2 } from "@/types/sqlReview";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const reviewTemplate: SQLReviewPolicyTemplateV2 & {
  review: { name: string; resources: string[] };
} = {
  id: "reviews/example",
  ruleList: [],
  review: {
    name: "Existing policy",
    resources: ["environments/test"],
  },
};

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  useVueState: vi.fn((getter: () => unknown) => getter()),
  rulesToTemplate: vi.fn(() => reviewTemplate),
  useProjectV1Store: vi.fn(),
  useSQLReviewStore: vi.fn(),
}));

let TemplateSelector: typeof import("./TemplateSelector").TemplateSelector;

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/react/lib/sql-review/utils", () => ({
  rulesToTemplate: mocks.rulesToTemplate,
}));

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) => (
    <span>{environmentName}</span>
  ),
}));

vi.mock("@/types", () => ({
  TEMPLATE_LIST_V2: [
    {
      id: "builtin/default",
      ruleList: [],
    },
  ],
}));

vi.mock("@/store", () => ({
  useProjectV1Store: mocks.useProjectV1Store,
  useSQLReviewStore: mocks.useSQLReviewStore,
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(async () => {
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.useVueState.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
  mocks.rulesToTemplate.mockReset();
  mocks.rulesToTemplate.mockReturnValue(reviewTemplate);
  mocks.useProjectV1Store.mockReset();
  mocks.useProjectV1Store.mockReturnValue({
    getOrFetchProjectByName: vi.fn(),
    getProjectByName: vi.fn(() => undefined),
  });
  mocks.useSQLReviewStore.mockReset();
  mocks.useSQLReviewStore.mockReturnValue({
    reviewPolicyList: [{ id: "policy-1" }],
    fetchReviewPolicyList: vi.fn(),
  });
  ({ TemplateSelector } = await import("./TemplateSelector"));
});

describe("TemplateSelector", () => {
  test("renders review templates as semantic buttons", () => {
    const onSelectTemplate = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <TemplateSelector onSelectTemplate={onSelectTemplate} />
    );

    render();

    const buttons = [...container.querySelectorAll("button")];
    const reviewButton = buttons.find((button) =>
      button.textContent?.includes("Existing policy")
    );

    expect(reviewButton).toBeTruthy();

    act(() => {
      reviewButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(onSelectTemplate).toHaveBeenCalledWith(reviewTemplate);

    unmount();
  });
});
