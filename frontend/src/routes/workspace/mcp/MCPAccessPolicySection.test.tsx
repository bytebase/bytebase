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
  // The disclosure remembers itself per browser, so each case starts from the
  // default rather than from whatever the previous one left open.
  localStorage.clear();
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
    expect(container.textContent).toContain(
      "settings.mcp.policy.mode.read-only.title"
    );
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
    const pending = deferred<undefined>();
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

  // The cards used to carry three "Best for" lines and three descriptions,
  // which is what made the edit state 375 words. One line, for the pick.
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
    // A SERVING mode, so the masking switch is on screen and inside the set
    // this asserts on. Picking Disabled here would withhold it (D7) and the
    // test would silently stop covering `disabled={saving}` on the Switch.
    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();

    // The Switch renders a span plus a hidden checkbox, and the checkbox is
    // what carries `disabled` — so the masking control is inside this set only
    // while a serving mode is picked. Pinned explicitly, because the set going
    // quietly back to radios-only is how this test stopped covering it before.
    const controls = () =>
      [
        ...container.querySelectorAll(
          'input[type="radio"], input[type="checkbox"]'
        ),
      ] as HTMLInputElement[];
    expect(
      container.querySelector('input[type="checkbox"]')
    ).not.toBeNull();
    expect(controls().length).toBeGreaterThan(3);
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
  test("the chip names the policy for assistive tech and carries the mode glyph", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    // Asserted on rendered text, not on an attribute: a Badge is a bare span,
    // whose implicit `generic` role ARIA forbids naming, so an aria-label here
    // would satisfy a DOM query while naming nothing for a screen reader.
    expect(container.textContent).toContain(
      "settings.mcp.policy.current(settings.mcp.policy.mode.read-only.title)"
    );
    expect(container.querySelector("span.sr-only")?.textContent).toBe(
      "settings.mcp.policy.current(settings.mcp.policy.mode.read-only.title)"
    );
    expect(
      container.querySelector('[aria-label^="settings.mcp.policy.current"]')
    ).toBeNull();
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
    mocks.serverInfo.value = {
      mcpSetting: {
        capability: MCPSetting_Capability.DISABLED,
        ignoreMaskingExemptions: false,
      },
    };
    mocks.loadServerInfo.mockResolvedValue(mocks.serverInfo.value);
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
      container.querySelector('[aria-label="settings.mcp.policy.masking.title"]')
    ).toBeNull();

    // Picking a serving mode again brings the control back.
    clickText(container, "settings.mcp.policy.mode.read-write.title");
    await flush();
    expect(
      container.querySelector('[aria-label="settings.mcp.policy.masking.title"]')
    ).not.toBeNull();
    unmount();
  });

  // Toggling masking and then picking Disabled would otherwise write a change
  // the admin can no longer see, because the control that made it is gone.
  test("saving Disabled leaves the stored masking flag untouched", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    const maskingSwitch = () =>
      container.querySelector(
        '[aria-label="settings.mcp.policy.masking.title"]'
      ) as HTMLElement | null;
    act(() => maskingSwitch()?.click());
    await flush();

    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();
    clickText(container, "settings.mcp.policy.save");
    await flush();

    // The mask is the whole statement of what this save writes: the masking
    // path is absent, so the stored flag is untouched whatever the body says.
    const request = mocks.upsertSetting.mock.calls.at(-1)?.[0];
    expect(request.updateMask.paths).toEqual(["value.mcp.capability"]);
    unmount();
  });

  // Nothing the editor would write, so Save stays disabled even though the
  // hidden draft differs from the stored flag.
  test("a masking draft alone cannot save under Disabled", async () => {
    mocks.serverInfo.value = {
      mcpSetting: {
        capability: MCPSetting_Capability.DISABLED,
        ignoreMaskingExemptions: false,
      },
    };
    mocks.loadServerInfo.mockResolvedValue(mocks.serverInfo.value);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();
    clickText(container, "settings.mcp.policy.edit");
    await flush();

    // Reach the toggle through a serving mode, set it, then return to Disabled.
    clickText(container, "settings.mcp.policy.mode.read-only.title");
    await flush();
    act(() =>
      (
        container.querySelector(
          '[aria-label="settings.mcp.policy.masking.title"]'
        ) as HTMLElement | null
      )?.click()
    );
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

  // The rule is "a serving mode admits a session", not "the mode is not
  // Disabled": with no mode picked, nothing is known to serve. Encoding it as a
  // negation let this state through, arming the unsaved-changes guard for a
  // change Save can never submit.
  test("the masking toggle is withheld when no mode is picked", async () => {
    mocks.serverInfo.value = {
      mcpSetting: {
        capability: MCPSetting_Capability.CAPABILITY_UNSPECIFIED,
        ignoreMaskingExemptions: false,
      },
    };
    mocks.loadServerInfo.mockResolvedValue(mocks.serverInfo.value);
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
      container.querySelector('[aria-label="settings.mcp.policy.masking.title"]')
    ).toBeNull();
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);
    unmount();
  });

  test("the masking badge is withheld on a disabled policy", async () => {
    mocks.serverInfo.value = {
      mcpSetting: {
        capability: MCPSetting_Capability.DISABLED,
        ignoreMaskingExemptions: true,
      },
    };
    mocks.loadServerInfo.mockResolvedValue(mocks.serverInfo.value);
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    expect(container.textContent).not.toContain(
      "settings.mcp.policy.masking.badge"
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
