import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

const mocks = vi.hoisted(() => ({
  captureFeatureGateMetric: vi.fn(),
  useAppStore: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useSubscriptionState: () => ({}),
}));

vi.mock("@/stores/app", () => ({ useAppStore: mocks.useAppStore }));

vi.mock("@/app/analytics/feature-gate", () => ({
  captureFeatureGateMetric: mocks.captureFeatureGateMetric,
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({ children, onClick }: React.ComponentProps<"a">) => (
    <a href="/setting/subscription" onClick={onClick}>
      {children}
    </a>
  ),
}));

vi.mock("./ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => children,
}));

let FeatureBadge: typeof import("./FeatureBadge").FeatureBadge;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  act(() => root.render(element));
  return { container, unmount: () => act(() => root.unmount()) };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useAppStore.mockImplementation(
    (selector: (state: Record<string, unknown>) => unknown) =>
      selector({
        hasInstanceFeature: () => false,
        instanceMissingLicense: () => false,
        getMinimumRequiredPlan: () => 2,
      })
  );
  ({ FeatureBadge } = await import("./FeatureBadge"));
});

describe("FeatureBadge", () => {
  test("captures clicks on a clickable locked badge", () => {
    const { container, unmount } = renderIntoContainer(
      <FeatureBadge feature={1} />
    );

    act(() => {
      container
        .querySelector("a")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(mocks.captureFeatureGateMetric).toHaveBeenCalledWith(
      "locked feature clicked",
      1,
      undefined
    );
    unmount();
  });
});
