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

const storePolicy = (
  capability: MCPSetting_Capability,
  ignoreMaskingExemptions = false
) => {
  mocks.serverInfo.value = { mcpSetting: { capability, ignoreMaskingExemptions } };
  mocks.loadServerInfo.mockResolvedValue(mocks.serverInfo.value);
  mocks.refreshServerInfo.mockResolvedValue(mocks.serverInfo.value);
};

const maskingBadgeText = (container: HTMLElement) =>
  [...container.querySelectorAll("span")].find((node) =>
    node.textContent?.startsWith("settings.mcp.policy.masking.badge")
  )?.textContent;

const maskingSwitch = (container: HTMLElement) =>
  container.querySelector(
    '[aria-label="settings.mcp.policy.masking.title"]'
  ) as HTMLElement | null;

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
  // The disclosure remembers itself per browser, so each case starts from the
  // default rather than from whatever the previous one left open.
  localStorage.clear();
  mocks.permissionDisabled.value = false;
  mocks.dataMaskingAvailable.value = true;
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

  test("the Best for line is shown once and follows the selection", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

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
    // A SERVING mode, so the masking switch is on screen and inside the set
    // this asserts on. Picking Disabled here would withhold it (D7) and the
    // test would silently stop covering `disabled={saving}` on the Switch.
    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();

    // The Switch renders a span plus a hidden checkbox, and the checkbox is
    // what carries `disabled`, so the masking control is in this set only while
    // a serving mode is picked: three radios and that one checkbox.
    const controls = () =>
      [
        ...container.querySelectorAll(
          'input[type="radio"], input[type="checkbox"]'
        ),
      ] as HTMLInputElement[];
    expect(controls()).toHaveLength(4);
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

  test("the open and details state survive leaving the page", async () => {
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
    expect(second.container.textContent).toContain(
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

  // The toggle governs what an MCP session may unmask, and Disabled admits
  // none, so offering it there would put a live control under a red line
  // saying no session can connect.
  test("the masking toggle is withheld while Disabled is picked", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(container.textContent).toContain("settings.mcp.policy.masking.title");

    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();
    expect(
      maskingSwitch(container)
    ).toBeNull();

    // Picking a serving mode again brings the control back.
    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();
    expect(
      maskingSwitch(container)
    ).not.toBeNull();
    unmount();
  });

  // The flag is inert under Disabled but still stored, so an explicit choice is
  // kept for when MCP is turned back on rather than quietly rolled back — and
  // the footer says so, since the control that made it is off screen.
  test("saving Disabled still carries a masking edit the admin made", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    act(() => maskingSwitch(container)?.click());
    await flush();

    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();
    expect(container.textContent).toContain(
      "settings.mcp.policy.masking-pending.ignored"
    );

    clickText(container, "settings.mcp.policy.save");
    await flush();

    const request = mocks.upsertSetting.mock.calls.at(-1)?.[0];
    expect(request.updateMask.paths).toEqual([
      "value.mcp.capability",
      "value.mcp.ignore_masking_exemptions",
    ]);
    expect(request.value.value.value.ignoreMaskingExemptions).toBe(true);
    unmount();
  });

  // Picking back to the stored mode with no other edit leaves nothing to write.
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

  test("a masking change reaches the update mask", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    act(() => maskingSwitch(container)?.click());
    await flush();
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);

    clickText(container, "settings.mcp.policy.save");
    await flush();

    const request = mocks.upsertSetting.mock.calls.at(-1)?.[0];
    expect(request.updateMask.paths).toEqual([
      "value.mcp.ignore_masking_exemptions",
    ]);
    expect(request.value.value.value.ignoreMaskingExemptions).toBe(true);
    unmount();
  });

  // Clicking through the modes to read their descriptions must not undo an
  // unrelated edit. Withholding the control where it governs nothing is not a
  // reason to discard what the admin set with it.
  test("the masking draft survives a detour through Disabled", async () => {
    storePolicy(MCPSetting_Capability.READ_WRITE, true);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    act(() => maskingSwitch(container)?.click());
    await flush();
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);

    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();
    // The control is withheld — it governs nothing under Disabled — but the
    // draft behind it is still the admin's, and the footer names the value Save
    // will write rather than the one that is stored.
    expect(maskingSwitch(container)).toBeNull();
    expect(container.textContent).toContain(
      "settings.mcp.policy.masking-pending.applied"
    );
    expect(container.textContent).not.toContain(
      "settings.mcp.policy.masking-pending.ignored"
    );
    expect(maskingBadgeText(container)).toBeUndefined();

    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();
    expect(maskingSwitch(container)?.getAttribute("aria-checked")).toBe("false");

    clickText(container, "settings.mcp.policy.save");
    await flush();
    const request = mocks.upsertSetting.mock.calls.at(-1)?.[0];
    expect(request.updateMask.paths).toEqual([
      "value.mcp.ignore_masking_exemptions",
    ]);
    expect(request.value.value.value.ignoreMaskingExemptions).toBe(false);
    unmount();
  });

  // An admin who cannot see the control cannot see the value either, so the
  // disclosure follows the value being saved rather than the edit that set it.
  test("a stored flag nobody touched is still named under a pick that hides it", async () => {
    storePolicy(MCPSetting_Capability.READ_ONLY, true);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();
    expect(container.textContent).toContain(
      "settings.mcp.policy.masking-pending.ignored"
    );

    clickText(container, "settings.mcp.policy.save");
    await flush();
    const request = mocks.upsertSetting.mock.calls.at(-1)?.[0];
    expect(request.value.value.value.ignoreMaskingExemptions).toBe(true);
    unmount();
  });

  // The chip is the stored flag's disclosure and the toggle is the draft's.
  // Rendering the chip in the editor asserted a live restriction under a pick
  // that admits no session, and read the stored pair while Save wrote the
  // draft.
  test("the masking chip belongs to the view, never to the editor", async () => {
    storePolicy(MCPSetting_Capability.READ_ONLY, true);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    expect(maskingBadgeText(container)).toBe("settings.mcp.policy.masking.badge");

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(maskingBadgeText(container)).toBeUndefined();

    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();
    expect(maskingBadgeText(container)).toBeUndefined();
    unmount();
  });

  // "MCP is off" is a claim about a mode somebody chose. A ceiling this build
  // cannot parse is failing closed instead, which has a different remedy.
  test("an unreadable ceiling is never reported as MCP being off", async () => {
    storePolicy(MCPSetting_Capability.CAPABILITY_UNSPECIFIED, true);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    expect(maskingBadgeText(container)).toBeUndefined();

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(maskingBadgeText(container)).toBeUndefined();
    expect(container.textContent).toContain("settings.mcp.policy.unreadable.pick");
    // "Pick a mode to save this policy" and "this policy will be saved" cannot
    // both hold. Nothing is pending until a mode is picked.
    expect(container.textContent).not.toContain(
      "settings.mcp.policy.masking-pending"
    );
    unmount();
  });

  // Opening the editor is not an edit. A form that cannot be saved has no save
  // to describe, and the view's chip is where the stored flag is reported.
  test("a pristine editor promises no save", async () => {
    storePolicy(MCPSetting_Capability.DISABLED, true);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    expect(container.textContent).not.toContain(
      "settings.mcp.policy.masking-pending"
    );
    expect(
      [...container.querySelectorAll("button")].find((button) =>
        button.textContent?.includes("settings.mcp.policy.save")
      )
    ).toHaveProperty("disabled", true);
    unmount();
  });

  // The footer says what will be written, never when it takes effect — so it
  // does not vary with a masking license the way the view's chip does.
  test("the pending line reads the same on an unlicensed workspace", async () => {
    mocks.dataMaskingAvailable.value = false;
    storePolicy(MCPSetting_Capability.READ_ONLY, true);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    expect(maskingBadgeText(container)).toBe(
      "settings.mcp.policy.masking.badge-unlicensed"
    );

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();
    expect(container.textContent).toContain(
      "settings.mcp.policy.masking-pending.ignored"
    );
    unmount();
  });

  // With no mode picked, nothing is known to serve, so the control that
  // configures a session has nothing to configure.
  test("the masking toggle is withheld when no mode is picked", async () => {
    storePolicy(MCPSetting_Capability.CAPABILITY_UNSPECIFIED);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    expect(container.textContent).toContain(
      "settings.mcp.policy.unreadable.pick"
    );
    expect(
      maskingSwitch(container)
    ).toBeNull();
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);
    unmount();
  });

  test("the masking badge reports a stored restriction", async () => {
    storePolicy(MCPSetting_Capability.READ_ONLY, true);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    // Exact, not substring: the plain key is a prefix of both suffixed ones, so
    // `toContain` would pass for every state including the two that report the
    // flag as inert.
    expect(maskingBadgeText(container)).toBe("settings.mcp.policy.masking.badge");
    unmount();
  });

  // Still shown without a license — hiding it would leave a stored flag with
  // nowhere to see it — but it must not assert a restriction nothing applies.
  // The editor's "not licensed" note is a different branch, behind
  // bb.settings.set, which a reader of this page may never reach.
  test("the masking badge says so when masking is unlicensed", async () => {
    mocks.dataMaskingAvailable.value = false;
    storePolicy(MCPSetting_Capability.READ_ONLY, true);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    expect(maskingBadgeText(container)).toBe(
      "settings.mcp.policy.masking.badge-unlicensed"
    );
    unmount();
  });

  // Storable under Disabled, so it needs a view there rather than a hole: the
  // badge says the flag is set and that MCP is off, instead of asserting a
  // restriction on sessions that cannot exist.
  test("the masking badge says MCP is off on a disabled policy", async () => {
    storePolicy(MCPSetting_Capability.DISABLED, true);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(maskingBadgeText(container)).toBe(
      "settings.mcp.policy.masking.badge-disabled"
    );
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
    expect(container.textContent).not.toContain("settings.mcp.policy.audit");

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
