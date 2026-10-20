import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  GuideContextProvider,
  useSharedGuideContext,
} from "./GuideContextProvider";

const mocks = vi.hoisted(() => ({
  dismissed: false,
  route: { name: "workspace.landing", params: {} },
  useGuideContext: vi.fn(
    ({ dismissed }: { dismissed: boolean }) => ({
      context: { projectName: "projects/orders" },
      contextReady: !dismissed,
    })
  ),
}));

vi.mock("@/app/router", () => ({
  useCurrentRoute: () => mocks.route,
}));
vi.mock("@/hooks/useAppState", () => ({
  useIntroStateByKey: (key: string) =>
    key === "workspace-setup-guide.dismissed" && mocks.dismissed,
}));
vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: object) => unknown) =>
    selector({ workspaceSetupGuideEnabled: () => true }),
}));
vi.mock("./selection", () => ({
  readGuideWorkspaceUsage: () => undefined,
  readSelectedGuideScenarioId: () => undefined,
}));
vi.mock("./useGuideContext", () => ({
  useGuideContext: mocks.useGuideContext,
}));

function GuideConsumer({ label }: { label: string }) {
  const { context, contextReady } = useSharedGuideContext();
  return (
    <div>
      {label}: {contextReady ? context.projectName : "loading"}
    </div>
  );
}

describe("GuideContextProvider", () => {
  test("shares one guide context with the landing and bottom surfaces", () => {
    mocks.dismissed = false;
    mocks.useGuideContext.mockClear();
    const view = render(
      <GuideContextProvider>
        <GuideConsumer label="landing" />
        <GuideConsumer label="bottom" />
      </GuideContextProvider>
    );

    expect(screen.getByText("landing: projects/orders")).toBeVisible();
    expect(screen.getByText("bottom: projects/orders")).toBeVisible();
    expect(mocks.useGuideContext).toHaveBeenCalledTimes(1);

    mocks.dismissed = true;
    view.rerender(
      <GuideContextProvider>
        <GuideConsumer label="landing" />
        <GuideConsumer label="bottom" />
      </GuideContextProvider>
    );

    expect(screen.getByText("landing: loading")).toBeVisible();
    expect(screen.getByText("bottom: loading")).toBeVisible();
    expect(mocks.useGuideContext).toHaveBeenCalledTimes(2);
    expect(mocks.useGuideContext).toHaveBeenLastCalledWith(
      expect.objectContaining({ dismissed: true, route: mocks.route })
    );
  });
});
