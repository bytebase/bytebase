import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { ProjectInstanceDetailPage } from "./ProjectInstanceDetailPage";

const mocks = vi.hoisted(() => ({
  viewProps: undefined as Record<string, unknown> | undefined,
  defaultProject: "",
  project: undefined as unknown,
  replace: vi.fn(),
}));

vi.mock("@/app/router", () => ({
  router: { replace: mocks.replace },
}));

vi.mock("@/components/instance/InstanceDetailView", () => ({
  InstanceDetailView: (props: Record<string, unknown>) => {
    mocks.viewProps = props;
    return <div />;
  },
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

describe("ProjectInstanceDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.viewProps = undefined;
    mocks.defaultProject = "";
    mocks.project = create(ProjectSchema, {
      name: "projects/app",
      title: "App",
    });
  });

  test("renders the nested instance with project ownership", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() =>
      root.render(
        <ProjectInstanceDetailPage projectId="app" instanceId="prod" />
      )
    );

    expect(mocks.viewProps).toMatchObject({
      instanceName: "projects/app/instances/prod",
      project: mocks.project,
    });

    const onLeave = mocks.viewProps?.onLeave as () => void;
    act(() => onLeave());
    expect(mocks.replace).toHaveBeenCalledWith({
      name: "workspace.project.instance",
      params: { projectId: "app" },
    });

    act(() => root.unmount());
  });

  test("redirects the default project", () => {
    mocks.defaultProject = "projects/app";
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() =>
      root.render(
        <ProjectInstanceDetailPage projectId="app" instanceId="prod" />
      )
    );

    expect(mocks.replace).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "app" },
    });
    expect(mocks.viewProps).toBeUndefined();

    act(() => root.unmount());
  });
});
