import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { InstanceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { ProjectCreateInstancePage } from "./ProjectCreateInstancePage";

const mocks = vi.hoisted(() => ({
  viewProps: undefined as Record<string, unknown> | undefined,
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

vi.mock("@/components/instance/CreateInstanceView", () => ({
  CreateInstanceView: (props: Record<string, unknown>) => {
    mocks.viewProps = props;
    return <div data-testid="create-instance-view" />;
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

const renderPage = async () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  await act(async () => {
    root.render(
      <ProjectCreateInstancePage projectId="app" />
    );
  });
  return { root };
};

describe("ProjectCreateInstancePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.viewProps = undefined;
    mocks.defaultProject = "";
    mocks.project = create(ProjectSchema, {
      name: "projects/app",
      title: "App",
    });
  });

  test("provides project ownership to the shared create view", async () => {
    const page = await renderPage();

    expect(mocks.viewProps).toMatchObject({
      parent: "projects/app",
      project: mocks.project,
    });

    act(() => page.root.unmount());
  });

  test("opens the nested detail route after creation", async () => {
    const page = await renderPage();
    const onCreated = mocks.viewProps?.onCreated as (instance: unknown) => void;

    act(() => {
      onCreated(
        create(InstanceSchema, {
          name: "projects/app/instances/prod",
          title: "Production",
        })
      );
    });

    expect(mocks.push).toHaveBeenCalledWith({
      name: "workspace.project.instance.detail",
      params: { projectId: "app", instanceId: "prod" },
      query: { syncingInstance: "prod" },
      hash: "databases",
    });

    act(() => page.root.unmount());
  });

  test("returns to the project instance page", async () => {
    const page = await renderPage();
    const onDismiss = mocks.viewProps?.onDismiss as () => void;

    act(() => onDismiss());

    expect(mocks.push).toHaveBeenCalledWith({
      name: "workspace.project.instance",
      params: { projectId: "app" },
    });

    act(() => page.root.unmount());
  });

  test("redirects the default project before rendering the form", async () => {
    mocks.defaultProject = "projects/app";
    const page = await renderPage();

    expect(mocks.replace).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "app" },
    });
    expect(mocks.viewProps).toBeUndefined();

    act(() => page.root.unmount());
  });
});
