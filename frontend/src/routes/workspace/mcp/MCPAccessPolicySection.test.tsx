import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useUnsavedChangesGuard: vi.fn(),
  upsertSetting: vi.fn(),
  loadServerInfo: vi.fn(),
  refreshServerInfo: vi.fn(),
  serverInfo: {
    value: {
      mcpSetting: {
        capability: 3,
        ignoreMaskingExemptions: false,
      },
    } as { mcpSetting?: { capability: MCPSetting_Capability; ignoreMaskingExemptions: boolean } } | undefined,
  },
  dataMaskingAvailable: { value: true },
  permissionDisabled: { value: false },
  permissionGuard: vi.fn(),
}));

vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: mocks.useUnsavedChangesGuard,
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({
    permissions,
    children,
  }: {
    permissions: string[];
    children: (props: { disabled: boolean }) => ReactElement;
  }) => {
    mocks.permissionGuard(permissions);
    return children({ disabled: mocks.permissionDisabled.value });
  },
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("@/stores/app", () => {
  const state = {
    upsertSetting: mocks.upsertSetting,
    loadServerInfo: mocks.loadServerInfo,
    refreshServerInfo: mocks.refreshServerInfo,
    hasFeature: () => mocks.dataMaskingAvailable.value,
    get serverInfo() {
      return mocks.serverInfo.value;
    },
  };
  const useAppStore = (selector: (s: unknown) => unknown) => selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

let MCPAccessPolicySection: typeof import("./MCPAccessPolicySection").MCPAccessPolicySection;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: () => act(() => root.render(element)),
    unmount: () => act(() => root.unmount()),
  };
};

const flush = () =>
  act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });

const deferred = <T,>() => {
  let resolve!: (v: T) => void;
  let reject!: (e: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
};

const clickText = (container: HTMLElement, text: string) => {
  const el = [...container.querySelectorAll("button, label")].find((n) =>
    n.textContent?.includes(text)
  );
  act(() => {
    (el as HTMLElement)?.click();
  });
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.permissionDisabled.value = false;
  mocks.dataMaskingAvailable.value = true;
  mocks.serverInfo.value = {
    mcpSetting: {
      capability: MCPSetting_Capability.READ_ONLY,
      ignoreMaskingExemptions: false,
    },
  };
  mocks.loadServerInfo.mockResolvedValue(mocks.serverInfo.value);
  mocks.refreshServerInfo.mockResolvedValue(mocks.serverInfo.value);
  mocks.upsertSetting.mockResolvedValue(undefined);
  ({ MCPAccessPolicySection } = await import("./MCPAccessPolicySection"));
});

describe("MCPAccessPolicySection", () => {
  test("reads the displayed policy from cached actuator info", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(mocks.loadServerInfo).toHaveBeenCalledOnce();
    expect(container.textContent).toContain("settings.mcp.policy.in-force");
    unmount();
  });

  // Codex raised exactly this on #21236 after the form moved off GeneralPage,
  // where it had been registered in the guarded section refs. Without an
  // assertion the regression returns silently, which is why both sibling forms
  // pin the call the same way (CreateInstanceView.test.tsx, ReviewCreation.test.tsx).
  test("registers unsaved edits with the navigation guard", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    // Picking a different ceiling is the unsaved edit that must be guarded.
    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);

    // Cancelling drops the edit, so the guard must stand down again.
    clickText(container, "common.cancel");
    await flush();
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    unmount();
  });

  test("shows the policy-read failure instead of a stale policy", async () => {
    mocks.serverInfo.value = undefined;
    mocks.loadServerInfo.mockResolvedValue(undefined);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(container.textContent).toContain(
      "settings.mcp.policy.read-failed.title"
    );
    expect(container.textContent).not.toContain("settings.mcp.policy.in-force");
    unmount();
  });

  test("uses the permission wrapper to disable policy editing", async () => {
    mocks.permissionDisabled.value = true;
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(mocks.permissionGuard).toHaveBeenCalledWith(["bb.settings.set"]);
    expect(
      [...container.querySelectorAll("button")].find((button) =>
        button.textContent?.includes("settings.mcp.policy.edit")
      )
    ).toHaveProperty("disabled", true);

    unmount();
  });

  test("repairs an unspecified capability reported by actuator info", async () => {
    mocks.serverInfo.value = {
      mcpSetting: {
        capability: MCPSetting_Capability.CAPABILITY_UNSPECIFIED,
        ignoreMaskingExemptions: false,
      },
    };
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(container.textContent).toContain(
      "settings.mcp.policy.unreadable.title"
    );
    expect(container.textContent).not.toContain("settings.mcp.policy.in-force");

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(container.textContent).toContain(
      "settings.mcp.policy.unreadable.pick"
    );

    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();
    clickText(container, "settings.mcp.policy.save");
    await flush();

    expect(mocks.upsertSetting).toHaveBeenCalledWith(
      expect.objectContaining({
        updateMask: expect.objectContaining({
          paths: ["value.mcp.capability"],
        }),
      })
    );
    const request = mocks.upsertSetting.mock.calls.at(-1)?.[0];
    expect(request.value.value.value.capability).toBe(4);

    unmount();
  });

  test("waits for actuator info before offering policy editing", async () => {
    const pending = deferred<undefined>();
    mocks.serverInfo.value = undefined;
    mocks.loadServerInfo.mockReturnValue(pending.promise);

    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(container.textContent).toContain("settings.mcp.policy.loading");
    expect(container.textContent).not.toContain("settings.mcp.policy.in-force");
    expect(container.textContent).not.toContain("settings.mcp.policy.edit");

    unmount();
  });

  test("says masking is unlicensed", async () => {
    mocks.dataMaskingAvailable.value = false;

    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    expect(container.textContent).toContain(
      "settings.mcp.policy.masking.unavailable"
    );
    unmount();
  });

  test("keeps policy-card content compact", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    const bestForLines = [...container.querySelectorAll("p")].filter((line) =>
      line.textContent?.includes(".best-for")
    );
    expect(bestForLines).toHaveLength(3);
    expect(bestForLines.every((line) => !line.classList.contains("mt-auto"))).toBe(
      true
    );
    unmount();
  });

  test("keeps the masking switch intrinsic", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    const maskingSwitch = container.querySelector(
      '[aria-label="settings.mcp.policy.masking.title"]'
    );
    expect(maskingSwitch?.classList.contains("shrink-0")).toBe(true);
    unmount();
  });

  // Codex, #21236: setSaving gated only the footer buttons. The request has
  // already captured pick and ignoreMasking, so a card clicked after Save went
  // out changed the visible draft and nothing else — then the success path
  // closed the editor and the click was gone, with no sign it had been dropped.
  test("the policy inputs are locked while a save is in flight", async () => {
    const inFlight = deferred<undefined>();
    mocks.upsertSetting.mockReturnValue(inFlight.promise);

    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();

    const controls = () => [
      ...container.querySelectorAll('input[type="radio"], input[type="checkbox"]'),
    ] as HTMLInputElement[];
    expect(controls().length).toBeGreaterThan(0);
    expect(controls().every((c) => !c.disabled)).toBe(true);

    clickText(container, "settings.mcp.policy.save");
    await flush();

    // Still open, still showing the draft — and now untouchable.
    expect(controls().length).toBeGreaterThan(0);
    expect(controls().every((c) => c.disabled)).toBe(true);

    act(() => inFlight.resolve(undefined));
    await flush();
    expect(mocks.refreshServerInfo).toHaveBeenCalledOnce();
    unmount();
  });

});
