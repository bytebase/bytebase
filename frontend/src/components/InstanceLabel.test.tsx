import type { MouseEventHandler, ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  instancesByName: {} as Record<string, { name: string; title: string }>,
  fetchInstance: vi.fn(),
  hasWorkspacePermissionV2: vi.fn((_permission: string) => true),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/EngineIcon", () => ({
  EngineIcon: ({ engine }: { engine: Engine }) => (
    <span data-engine={Engine[engine]} data-testid="engine-icon" />
  ),
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({
    children,
    className,
    onClick,
    to,
  }: {
    children: ReactNode;
    className?: string;
    onClick?: MouseEventHandler<HTMLAnchorElement>;
    to: { name?: string; params?: Record<string, string> };
  }) => (
    <a
      className={className}
      data-instance-id={to.params?.instanceId}
      data-route-name={to.name}
      onClick={onClick}
    >
      {children}
    </a>
  ),
}));

vi.mock("@/stores/app", () => {
  const appState = {
    get instancesByName() {
      return mocks.instancesByName;
    },
  };
  const useAppStore = (selector: (state: typeof appState) => unknown) =>
    selector(appState);
  useAppStore.getState = () => ({
    fetchInstance: mocks.fetchInstance,
  });
  return { useAppStore };
});

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

import { InstanceLabel } from "./InstanceLabel";

const render = async (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  await act(async () => {
    root.render(element);
    await Promise.resolve();
  });

  return { container, root };
};

describe("InstanceLabel", () => {
  let root: Root | undefined;

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.hasWorkspacePermissionV2.mockReturnValue(true);
    mocks.instancesByName = {
      "instances/prod": {
        name: "instances/prod",
        title: "Prod Instance",
        engine: Engine.POSTGRES,
        activation: true,
        dataSources: [],
      } as never,
    };
  });

  afterEach(() => {
    act(() => root?.unmount());
    document.body.innerHTML = "";
    root = undefined;
  });

  test("renders engine icon and instance title as a label", async () => {
    const rendered = await render(<InstanceLabel instanceName="instances/prod" />);
    root = rendered.root;

    expect(rendered.container.querySelector("a")).toBeNull();
    expect(rendered.container.querySelector("[data-testid='engine-icon']"))
      .not.toBeNull();
    expect(rendered.container.textContent).toBe("Prod Instance");
    expect(mocks.fetchInstance).toHaveBeenCalledWith("instances/prod");
  });

  test("renders a provided instance resource without fetching", async () => {
    const rendered = await render(
      <InstanceLabel
        instance={{
          name: "instances/reporting",
          title: "Reporting",
          engine: Engine.MYSQL,
          activation: true,
          dataSources: [],
        } as never}
      />
    );
    root = rendered.root;

    expect(rendered.container.querySelector("[data-testid='engine-icon']"))
      .not.toBeNull();
    expect(rendered.container.textContent).toBe("Reporting");
    expect(mocks.fetchInstance).not.toHaveBeenCalled();
  });

  test("renders engine icon and instance title as a link", async () => {
    const rendered = await render(
      <InstanceLabel instanceName="instances/prod" link />
    );
    root = rendered.root;

    const link = rendered.container.querySelector("a");
    expect(link?.textContent).toBe("Prod Instance");
    expect(link?.getAttribute("data-route-name")).toBe(
      "workspace.instance.detail"
    );
    expect(link?.getAttribute("data-instance-id")).toBe("prod");
  });

  test("renders invalid instance names as plain text even when link is requested", async () => {
    const rendered = await render(<InstanceLabel instanceName="instances/" link />);
    root = rendered.root;

    expect(rendered.container.querySelector("a")).toBeNull();
    expect(rendered.container.querySelector("[data-testid='engine-icon']")).toBeNull();
    expect(rendered.container.textContent).toBe("instances/");
    expect(mocks.fetchInstance).not.toHaveBeenCalled();
  });

  test("renders the instance resource name as plain text without instance get permission", async () => {
    mocks.hasWorkspacePermissionV2.mockImplementation(
      (permission) => permission !== "bb.instances.get"
    );

    const rendered = await render(
      <InstanceLabel instanceName="instances/prod" link />
    );
    root = rendered.root;

    expect(rendered.container.querySelector("a")).toBeNull();
    expect(rendered.container.querySelector("[data-testid='engine-icon']")).toBeNull();
    expect(rendered.container.textContent).toBe("instances/prod");
    expect(mocks.fetchInstance).not.toHaveBeenCalled();
  });
});
