import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { InstanceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { CreateInstancePage } from "./CreateInstancePage";

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  viewProps: undefined as Record<string, unknown> | undefined,
}));

vi.mock("@/app/router", () => ({
  router: { push: mocks.push },
}));

vi.mock("@/components/instance/CreateInstanceView", () => ({
  CreateInstanceView: (props: Record<string, unknown>) => {
    mocks.viewProps = props;
    return <div data-testid="create-instance-view" />;
  },
}));

describe("CreateInstancePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.viewProps = undefined;
  });

  test("renders the workspace-owned create view", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => root.render(<CreateInstancePage />));

    expect(mocks.viewProps).not.toHaveProperty("parent");
    act(() => root.unmount());
  });

  test("opens the workspace instance detail after creation", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => root.render(<CreateInstancePage />));

    const onCreated = mocks.viewProps?.onCreated as (instance: unknown) => void;
    act(() => {
      onCreated(
        create(InstanceSchema, {
          name: "instances/prod",
          title: "Production",
        })
      );
    });

    expect(mocks.push).toHaveBeenCalledWith({
      name: "workspace.instance.detail",
      params: { instanceId: "prod" },
      query: {
        syncingInstance: "prod",
        intro: "prepare-database",
        tip: "transfer-databases-to-project",
      },
      hash: "databases",
    });

    act(() => root.unmount());
  });
});
