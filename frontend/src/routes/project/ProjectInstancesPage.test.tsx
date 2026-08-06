import { create } from "@bufbuild/protobuf";
import type { ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  PROJECT_V1_ROUTE_INSTANCE_CREATE,
  PROJECT_V1_ROUTE_INSTANCE_DETAIL,
  PROJECT_V1_ROUTE_INSTANCES,
} from "@/app/router/handles";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { ProjectInstancesPage } from "./ProjectInstancesPage";

const mocks = vi.hoisted(() => ({
  dashboardProps: undefined as Record<string, unknown> | undefined,
  defaultProject: "",
  project: undefined as unknown,
  push: vi.fn(),
  replace: vi.fn(),
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.push,
    replace: mocks.replace,
  },
}));

vi.mock("@/components/instance/InstanceDashboard", () => ({
  InstanceDashboard: (props: Record<string, unknown>) => {
    mocks.dashboardProps = props;
    return <div data-testid="instance-dashboard" />;
  },
}));

vi.mock("@/components/ProjectPageLayout", () => ({
  ProjectPageLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      projectsByName: mocks.project
        ? { "projects/app": mocks.project }
        : {},
      serverInfo: { defaultProject: mocks.defaultProject },
    }),
}));

const renderPage = async () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  await act(async () => {
    root.render(<ProjectInstancesPage projectId="app" />);
  });
  return { container, root };
};

describe("ProjectInstancesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.dashboardProps = undefined;
    mocks.defaultProject = "";
    mocks.project = create(ProjectSchema, {
      name: "projects/app",
      title: "App",
    });
  });

  test("defines project instance route handles", () => {
    expect(PROJECT_V1_ROUTE_INSTANCES).toBe("workspace.project.instance");
    expect(PROJECT_V1_ROUTE_INSTANCE_CREATE).toBe(
      "workspace.project.instance.create"
    );
    expect(PROJECT_V1_ROUTE_INSTANCE_DETAIL).toBe(
      "workspace.project.instance.detail"
    );
  });

  test("renders the shared dashboard with the exact project parent", async () => {
    const page = await renderPage();

    expect(mocks.dashboardProps).toMatchObject({
      parent: "projects/app",
      project: mocks.project,
      layout: "project",
    });

    act(() => page.root.unmount());
  });

  test("redirects the default project before rendering the dashboard", async () => {
    mocks.defaultProject = "projects/app";
    const page = await renderPage();

    expect(mocks.replace).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "app" },
    });
    expect(mocks.dashboardProps).toBeUndefined();

    act(() => page.root.unmount());
  });
});
