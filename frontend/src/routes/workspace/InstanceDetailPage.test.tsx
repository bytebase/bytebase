import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { InstanceDetailPage } from "./InstanceDetailPage";

const mocks = vi.hoisted(() => ({
  viewProps: undefined as Record<string, unknown> | undefined,
}));

vi.mock("@/components/instance/InstanceDetailView", () => ({
  InstanceDetailView: (props: Record<string, unknown>) => {
    mocks.viewProps = props;
    return <div />;
  },
}));

describe("InstanceDetailPage", () => {
  test("renders a workspace-owned instance", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => root.render(<InstanceDetailPage instanceId="prod" />));

    expect(mocks.viewProps).toEqual({ instanceName: "instances/prod" });
    act(() => root.unmount());
  });
});
