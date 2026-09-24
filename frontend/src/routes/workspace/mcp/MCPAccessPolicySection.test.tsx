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
      mcpSetting: { capability: 3 },
    } as { mcpSetting?: { capability: MCPSetting_Capability } } | undefined,
  },
  permissionDisabled: { value: false },
  permissionGuard: vi.fn(),
  pushNotification: vi.fn(),
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

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));

vi.mock("@/stores/app", () => {
  const state = {
    upsertSetting: mocks.upsertSetting,
    loadServerInfo: mocks.loadServerInfo,
    refreshServerInfo: mocks.refreshServerInfo,
    get serverInfo() {
      return mocks.serverInfo.value;
    },
  };
  const useAppStore = (selector: (s: unknown) => unknown) => selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, string>) =>
      vars ? `${key}(${Object.values(vars).join(",")})` : key,
  }),
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

const storePolicy = (capability: MCPSetting_Capability) => {
  mocks.serverInfo.value = { mcpSetting: { capability } };
  mocks.loadServerInfo.mockResolvedValue(mocks.serverInfo.value);
  mocks.refreshServerInfo.mockResolvedValue(mocks.serverInfo.value);
};

const clickText = (container: HTMLElement, text: string) => {
  const el = [...container.querySelectorAll("button, label")].find((n) =>
    n.textContent?.includes(text)
  );
  act(() => {
    (el as HTMLElement)?.click();
  });
};

const selectCapability = (container: HTMLElement, value: number) => {
  const capabilities = [
    MCPSetting_Capability.DISABLED,
    MCPSetting_Capability.READ_ONLY,
    MCPSetting_Capability.READ_WRITE,
  ];
  const radio = container.querySelectorAll<HTMLElement>('[role="radio"]')[
    capabilities.indexOf(value)
  ];
  expect(radio).toBeTruthy();
  act(() => {
    radio!.click();
  });
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.permissionDisabled.value = false;
  storePolicy(MCPSetting_Capability.READ_ONLY);
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
    expect(container.textContent).toContain(
      "settings.mcp.policy.mode.read-only.title"
    );
    expect(container.textContent).toContain(
      "settings.mcp.policy.description(settings.mcp.policy.bound,settings.mcp.policy.audit)"
    );
    unmount();
  });

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
    selectCapability(container, MCPSetting_Capability.DISABLED);
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
    expect(container.textContent).not.toContain(
      "settings.mcp.policy.mode.read-only.title"
    );
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
    storePolicy(MCPSetting_Capability.CAPABILITY_UNSPECIFIED);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(container.textContent).toContain(
      "settings.mcp.policy.unreadable.title"
    );
    expect(container.textContent).not.toContain(
      "settings.mcp.policy.mode.read-only.title"
    );

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(container.textContent).toContain(
      "settings.mcp.policy.unreadable.pick"
    );
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    selectCapability(container, MCPSetting_Capability.READ_WRITE);
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
    const pending = Promise.withResolvers<undefined>();
    mocks.serverInfo.value = undefined;
    mocks.loadServerInfo.mockReturnValue(pending.promise);

    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(container.textContent).toContain("settings.mcp.policy.loading");
    expect(container.textContent).not.toContain(
      "settings.mcp.policy.mode.read-only.title"
    );
    expect(container.textContent).not.toContain("settings.mcp.policy.edit");

    unmount();
  });

  test("the Best for line is shown once and follows the selection", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    const radios = container.querySelectorAll('[role="radio"]');
    expect(radios).toHaveLength(3);
    expect(radios[1]).toHaveAttribute("aria-checked", "true");

    const bestForLines = () =>
      [...container.querySelectorAll("p")].filter((line) =>
        line.textContent?.includes(".best-for")
      );
    expect(bestForLines()).toHaveLength(1);
    expect(bestForLines()[0]?.textContent).toBe(
      "settings.mcp.policy.mode.read-only.best-for"
    );

    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();
    expect(bestForLines()).toHaveLength(1);
    expect(bestForLines()[0]?.textContent).toBe(
      "settings.mcp.policy.mode.read-write.best-for"
    );
    unmount();
  });

  test("the policy inputs are locked while a save is in flight", async () => {
    const inFlight = Promise.withResolvers<undefined>();
    mocks.upsertSetting.mockReturnValue(inFlight.promise);

    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();

    const controls = () =>
      [
        ...container.querySelectorAll('input[type="radio"]'),
      ] as HTMLInputElement[];
    expect(controls()).toHaveLength(3);
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

  // The chip is the subject of the view: it carries the mode's own glyph, so
  // the identity picked in the selector is the identity shown in force and, on
  // the consent page, the identity the person approving sees.
  test("the chip names the policy in force, not only the mode", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(container.textContent).toContain(
      "settings.mcp.policy.current(settings.mcp.policy.mode.read-only.title)"
    );
    unmount();
  });

  test("the disclosure is collapsed by default, opens, and follows the pick", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(container.textContent).toContain(
      "settings.mcp.ladder.summary.read-only"
    );
    expect(container.querySelectorAll("li")).toHaveLength(0);

    clickText(container, "settings.mcp.ladder.summary.read-only");
    await flush();
    expect(container.querySelectorAll("li").length).toBeGreaterThan(0);

    // The open state carries into editing, and the list follows the pick
    // rather than the stored mode.
    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(container.textContent).toContain(
      "settings.mcp.ladder.heading(settings.mcp.policy.mode.read-only.title)"
    );
    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();
    expect(container.textContent).toContain(
      "settings.mcp.ladder.heading(settings.mcp.policy.mode.read-write.title)"
    );
    expect(container.textContent).toContain("settings.mcp.ladder.tier.write");
    unmount();
  });

  test("the open and details state start over when the page is reopened", async () => {
    const first = renderIntoContainer(<MCPAccessPolicySection />);
    first.render();
    await flush();
    clickText(first.container, "settings.mcp.ladder.summary.read-only");
    await flush();
    clickText(first.container, "settings.mcp.ladder.show-details");
    await flush();
    expect(first.container.textContent).toContain(
      "settings.mcp.ladder.row.read-schemas.details"
    );
    first.unmount();

    const second = renderIntoContainer(<MCPAccessPolicySection />);
    second.render();
    await flush();
    expect(second.container.querySelectorAll("li")).toHaveLength(0);
    clickText(second.container, "settings.mcp.ladder.summary.read-only");
    await flush();
    expect(second.container.querySelectorAll("li").length).toBeGreaterThan(0);
    expect(second.container.textContent).not.toContain(
      "settings.mcp.ladder.row.read-schemas.details"
    );
    second.unmount();
  });

  // Disabled has no list, so red means "no capability" on both surfaces: the
  // plain sentence in view, the static line in the disclosure slot in edit.
  test("Disabled says its one sentence in view and its static line in edit", async () => {
    storePolicy(MCPSetting_Capability.DISABLED);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(container.textContent).toContain(
      "settings.mcp.policy.mode.disabled.description"
    );
    expect(container.textContent).not.toContain("settings.mcp.ladder.summary");

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(container.textContent).toContain("settings.mcp.ladder.disabled");
    expect(container.textContent).not.toContain("settings.mcp.ladder.heading");
    unmount();
  });

  // Picking back to the stored mode leaves nothing to write.
  test("returning to the stored mode with no edit cannot save", async () => {
    storePolicy(MCPSetting_Capability.DISABLED);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    clickText(container, "settings.mcp.policy.mode.read-only.title");
    await flush();
    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);
    expect(
      [...container.querySelectorAll("button")].find((button) =>
        button.textContent?.includes("settings.mcp.policy.save")
      )
    ).toHaveProperty("disabled", true);
    unmount();
  });

  // The write landed but the card can no longer read it back. Closing the
  // editor would present the pre-save policy as current, so the editor stays
  // open with the pick intact and saving again is the same write.
  test("a failed re-read keeps the editor open and claims no success", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();
    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();

    mocks.refreshServerInfo.mockRejectedValue(new Error("read failed"));
    clickText(container, "settings.mcp.policy.save");
    await flush();

    expect(mocks.upsertSetting).toHaveBeenCalledOnce();
    expect(mocks.pushNotification).not.toHaveBeenCalled();
    expect(container.textContent).toContain("settings.mcp.policy.save");
    expect(
      [...container.querySelectorAll("button")].find((button) =>
        button.textContent?.includes("settings.mcp.policy.save")
      )
    ).toHaveProperty("disabled", false);
    unmount();
  });

  test("a failed save closes nothing and reports no success", async () => {
    mocks.upsertSetting.mockRejectedValue(new Error("write failed"));
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();
    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();

    clickText(container, "settings.mcp.policy.save");
    await flush();

    expect(mocks.pushNotification).not.toHaveBeenCalled();
    expect(mocks.refreshServerInfo).not.toHaveBeenCalled();
    expect(container.textContent).toContain("settings.mcp.policy.save");
    unmount();
  });

  // Opening the editor is not an edit.
  test("a pristine editor promises no save", async () => {
    storePolicy(MCPSetting_Capability.DISABLED);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    expect(
      [...container.querySelectorAll("button")].find((button) =>
        button.textContent?.includes("settings.mcp.policy.save")
      )
    ).toHaveProperty("disabled", true);
    unmount();
  });

  // The tightening note is about a change being made, not about the current
  // state, so it belongs to the editor; the audit fact moved to the section
  // description and must not come back as a second line under the chip.
  test("the footer shows only while editing and names the change when dirty", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    expect(container.textContent).not.toContain("settings.mcp.policy.tightening");
    expect(container.textContent).toContain("settings.mcp.policy.audit");

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(container.textContent).toContain("settings.mcp.policy.tightening");
    expect(container.textContent).not.toContain(
      "settings.mcp.policy.tightening-change"
    );

    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();
    expect(container.textContent).toContain(
      "settings.mcp.policy.tightening-change(settings.mcp.policy.mode.read-only.title,settings.mcp.policy.mode.read-write.title)"
    );
    unmount();
  });
});
