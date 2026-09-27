import type { ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { InstancesPage } from "./InstancesPage";

const mocks = vi.hoisted(() => ({
  dashboardProps: undefined as Record<string, unknown> | undefined,
  push: vi.fn(),
}));

vi.mock("@/app/router", () => ({
  router: { push: mocks.push },
}));

vi.mock("@/components/instance/InstanceDashboard", () => ({
  InstanceDashboard: (props: Record<string, unknown>) => {
    mocks.dashboardProps = props;
    return <div data-testid="instance-dashboard" />;
  },
}));

vi.mock("@/components/WorkspacePageLayout", () => ({
  WorkspacePageLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

describe("InstancesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.dashboardProps = undefined;
  });

  test("renders the shared dashboard without a project parent", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstancesPage />);
    });

    expect(mocks.dashboardProps).toMatchObject({ layout: "workspace" });
    expect(mocks.dashboardProps).not.toHaveProperty("parent");

    act(() => root.unmount());
  });
});
