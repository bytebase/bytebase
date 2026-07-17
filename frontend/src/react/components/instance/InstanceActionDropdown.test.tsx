import { create } from "@bufbuild/protobuf";
import { fireEvent } from "@testing-library/react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import { InstanceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { InstanceActionDropdown } from "./InstanceActionDropdown";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

globalThis.ResizeObserver ??= class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

const mocks = vi.hoisted(() => ({
  hasWorkspacePermissionV2: vi.fn(() => true),
  archiveInstance: vi.fn(),
  restoreInstance: vi.fn(),
  deleteInstance: vi.fn(),
  routerReplace: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/react/router", () => ({
  router: {
    replace: mocks.routerReplace,
  },
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      archiveInstance: mocks.archiveInstance,
      restoreInstance: mocks.restoreInstance,
      deleteInstance: mocks.deleteInstance,
    }),
  },
}));

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/utils", async () => {
  const actual = await vi.importActual<typeof import("@/utils")>("@/utils");
  return {
    ...actual,
    hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
  };
});

const renderMenu = async (state: State) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  const instance = create(InstanceSchema, {
    name: "instances/prod",
    title: "Production",
    state,
  });

  await act(async () => {
    root.render(<InstanceActionDropdown instance={instance} />);
  });
  await act(async () => {
    container
      .querySelector("button")
      ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
  await vi.waitFor(() => {
    expect(document.body.querySelector("[role='menu']")).toBeTruthy();
  });

  return {
    root,
    text: () => document.body.textContent ?? "",
  };
};

describe("InstanceActionDropdown", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    document.body.innerHTML = "";
  });

  test("offers permanent delete for an active instance", async () => {
    const menu = await renderMenu(State.ACTIVE);

    expect(menu.text()).toContain("common.archive");
    expect(menu.text()).toContain("common.delete");

    act(() => menu.root.unmount());
  });

  test("offers permanent delete for an archived instance", async () => {
    const menu = await renderMenu(State.DELETED);

    expect(menu.text()).toContain("common.restore");
    expect(menu.text()).toContain("common.delete");

    act(() => menu.root.unmount());
  });

  test("opens the default instance list after archiving", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    mocks.archiveInstance.mockResolvedValue(undefined);
    const menu = await renderMenu(State.ACTIVE);
    const archiveItem = Array.from(
      document.body.querySelectorAll("[role='menuitem']")
    ).find((el) => el.textContent === "common.archive");

    await act(async () => {
      archiveItem?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      await Promise.resolve();
    });

    expect(mocks.archiveInstance).toHaveBeenCalled();
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "workspace.instance",
    });

    confirmSpy.mockRestore();
    act(() => menu.root.unmount());
  });

  test("archives before purging an active instance after resource id confirmation", async () => {
    mocks.archiveInstance.mockResolvedValue(undefined);
    mocks.deleteInstance.mockResolvedValue(undefined);
    const menu = await renderMenu(State.ACTIVE);
    const deleteItem = Array.from(
      document.body.querySelectorAll("[role='menuitem']")
    ).find((el) => el.textContent === "common.delete");

    await act(async () => {
      deleteItem?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const input = document.body.querySelector("input") as HTMLInputElement;
    const deleteButtons = Array.from(
      document.body.querySelectorAll("button")
    ).filter((el) => el.textContent === "common.delete");
    const dialogDeleteButton = deleteButtons.at(-1) as HTMLButtonElement;
    expect(dialogDeleteButton.disabled).toBe(true);

    await act(async () => {
      fireEvent.change(input, { target: { value: "prod" } });
    });
    expect(dialogDeleteButton.disabled).toBe(false);

    await act(async () => {
      dialogDeleteButton.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      await Promise.resolve();
    });

    expect(mocks.archiveInstance).toHaveBeenCalledWith(
      expect.objectContaining({ name: "instances/prod" }),
      true
    );
    expect(mocks.deleteInstance).toHaveBeenCalledWith("instances/prod");
    expect(mocks.archiveInstance.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.deleteInstance.mock.invocationCallOrder[0]
    );
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "workspace.instance",
    });

    act(() => menu.root.unmount());
  });

  test("purges an archived instance after resource id confirmation", async () => {
    mocks.deleteInstance.mockResolvedValue(undefined);
    const menu = await renderMenu(State.DELETED);
    const deleteItem = Array.from(
      document.body.querySelectorAll("[role='menuitem']")
    ).find((el) => el.textContent === "common.delete");

    await act(async () => {
      deleteItem?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const input = document.body.querySelector("input") as HTMLInputElement;
    const deleteButtons = Array.from(
      document.body.querySelectorAll("button")
    ).filter((el) => el.textContent === "common.delete");
    const dialogDeleteButton = deleteButtons.at(-1) as HTMLButtonElement;

    await act(async () => {
      fireEvent.change(input, { target: { value: "prod" } });
    });
    await act(async () => {
      dialogDeleteButton.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      await Promise.resolve();
    });

    expect(mocks.archiveInstance).not.toHaveBeenCalled();
    expect(mocks.deleteInstance).toHaveBeenCalledWith("instances/prod");

    act(() => menu.root.unmount());
  });
});
