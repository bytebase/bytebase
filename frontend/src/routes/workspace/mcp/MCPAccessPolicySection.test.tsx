import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useUnsavedChangesGuard: vi.fn(),
  getSetting: vi.fn(),
  getMCPInfo: vi.fn(),
  setSettingByName: vi.fn(),
  upsertSetting: vi.fn(),
  hasFeature: vi.fn(() => true),
  sheetProps: [] as Record<string, unknown>[],
  mcpSetting: {
    value: undefined as
      | {
          capability: number;
          ignoreMaskingExemptions: boolean;
        }
      | undefined,
  },
}));

vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: mocks.useUnsavedChangesGuard,
}));

vi.mock("@/api", () => ({
  settingServiceClientConnect: { getSetting: mocks.getSetting },
  workspaceServiceClientConnect: { getMCPInfo: mocks.getMCPInfo },
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({
    children,
  }: {
    children: (props: { disabled: boolean }) => ReactElement;
  }) => children({ disabled: false }),
}));

// Stubbed to record its props: the drawer is rendered unconditionally, so this
// captures what the section hands it on every render without opening it.
vi.mock("@/components/mcp/MCPModeContentsSheet", () => ({
  MCPModeContentsSheet: (props: Record<string, unknown>) => {
    mocks.sheetProps.push(props);
    return null;
  },
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("@/stores/app", () => {
  const state = {
    settingsByName: {},
    getSettingByName: () =>
      mocks.mcpSetting.value === undefined
        ? undefined
        : { value: { value: { case: "mcp", value: mocks.mcpSetting.value } } },
    setSettingByName: mocks.setSettingByName,
    upsertSetting: mocks.upsertSetting,
    hasFeature: mocks.hasFeature,
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

const toggleMasking = (container: HTMLElement) => {
  const input = container.querySelector('input[type="checkbox"]');
  act(() => {
    (input as HTMLInputElement).click();
  });
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
  mocks.sheetProps.length = 0;
  mocks.mcpSetting.value = {
    capability: 3, // READ_ONLY
    ignoreMaskingExemptions: false,
  };
  mocks.upsertSetting.mockResolvedValue(undefined);
  mocks.hasFeature.mockReturnValue(true);
  mocks.getSetting.mockResolvedValue({ name: "settings/MCP" });
  mocks.getMCPInfo.mockResolvedValue({
    capability: 3,
    ignoreMaskingExemptions: false,
    dataMaskingAvailable: true,
    modes: [],
    methods: [],
    engines: [],
  });
  ({ MCPAccessPolicySection } = await import("./MCPAccessPolicySection"));
});

describe("MCPAccessPolicySection", () => {
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

  // The read is deliberately uncached because the row changes out of band. When
  // it fails, the store may still hold a value from an earlier visit, and
  // rendering that reports a ceiling nobody is enforcing.
  test("a failed read outranks a value left in the store", async () => {
    mocks.getSetting.mockRejectedValue(new Error("does not parse"));
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

  // Codex, #21236: the drawer opens from a mode card the admin is choosing, so
  // it previews the candidate policy. Reading the persisted value described the
  // masking behavior they were about to replace.
  test("the drawer previews the draft masking value, not the stored one", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    expect(
      mocks.sheetProps.at(-1)?.ignoreMaskingExemptions
    ).toBe(false);

    toggleMasking(container);
    await flush();

    // Self-checking: if the click failed to flip the draft, the guard assertion
    // fails rather than the prop assertion passing for the wrong reason.
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);
    expect(mocks.sheetProps.at(-1)?.ignoreMaskingExemptions).toBe(true);
    expect(mocks.mcpSetting.value?.ignoreMaskingExemptions).toBe(false);

    unmount();
  });

  // Codex, #21236. Two halves of one race, both pinned here.
  test("waits for its own read before offering the cached policy for editing", async () => {
    // Second visit: the store already holds a value from visit one.
    const pending = deferred<{ name: string }>();
    mocks.getSetting.mockReturnValue(pending.promise);

    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    // The cached ceiling must not be presented as authoritative, and Edit must
    // not be reachable, while this mount's own read is still in flight.
    expect(container.textContent).toContain("settings.mcp.policy.loading");
    expect(container.textContent).not.toContain("settings.mcp.policy.in-force");
    expect(container.textContent).not.toContain("settings.mcp.policy.edit");

    act(() => pending.resolve({ name: "settings/MCP" }));
    await flush();
    expect(container.textContent).toContain("settings.mcp.policy.in-force");

    unmount();
  });

  test("a mode-contents read that lands after a save does not revert the page", async () => {
    // The setting read is gated, so the save path cannot start until it
    // answers. GetMCPInfo is not gated and genuinely fires twice — once on
    // mount, once after the save — so this is the interleaving that survives.
    const slowInfo = deferred<Record<string, unknown>>();
    mocks.getMCPInfo.mockReturnValueOnce(slowInfo.promise);
    const freshInfo = {
      capability: 1,
      ignoreMaskingExemptions: true,
      dataMaskingAvailable: true,
      modes: [],
      methods: [],
      engines: [],
    };
    mocks.getMCPInfo.mockResolvedValue(freshInfo);

    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    clickText(container, "settings.mcp.policy.edit");
    await flush();
    clickText(container, "settings.mcp.policy.mode.disabled.title");
    await flush();
    clickText(container, "settings.mcp.policy.save");
    await flush();

    // Now the mount's read finally answers, with what it captured beforehand.
    act(() =>
      slowInfo.resolve({
        capability: 4,
        ignoreMaskingExemptions: false,
        dataMaskingAvailable: false,
        modes: [],
        methods: [],
        engines: [],
      })
    );
    await flush();

    expect(mocks.sheetProps.at(-1)?.info).toBe(freshInfo);
    unmount();
  });

  // Codex, #21236: this warning used to be gated on GetMCPInfo, which refuses
  // outright under an unreadable or unserved ceiling (BOT-106) — the state an
  // admin is on this page to repair. Hiding it there lets them arm a toggle
  // that does nothing while believing they tightened masking.
  test("says masking is unlicensed even when the mode data fails", async () => {
    mocks.hasFeature.mockReturnValue(false);
    mocks.getMCPInfo.mockRejectedValue(new Error("the ceiling cannot be read"));

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

  // Codex, #21236: the drawer's info is a required prop now, so a pending or
  // refused GetMCPInfo cannot render as a mode that serves nothing — "0 of 0"
  // over the workspace whose ceiling the drawer exists to explain. GetMCPInfo
  // refuses outright under an unreadable ceiling (BOT-106), which is the state
  // an admin is on this page to repair.
  test("no mode-contents drawer while the mode data is missing", async () => {
    mocks.getMCPInfo.mockRejectedValue(new Error("the ceiling cannot be read"));

    const { container, render, unmount } = renderIntoContainer(
      <MCPAccessPolicySection />
    );
    render();
    await flush();

    // The policy itself still reads, so the page is up and editable.
    expect(container.textContent).toContain("settings.mcp.policy.in-force");
    expect(mocks.sheetProps).toHaveLength(0);

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
    unmount();
  });

  // The generations are per-mount; the setting store they write is the
  // application's. A read left flying by a visit the admin navigated away from
  // still passed its own check and wrote — which is Codex's corruption with
  // "the cached policy" replaced by "the previous visit's unfinished read".
  test("a read left in flight by an unmounted visit cannot write", async () => {
    const abandoned = deferred<{ name: string }>();
    mocks.getSetting.mockReturnValueOnce(abandoned.promise);

    const first = renderIntoContainer(<MCPAccessPolicySection />);
    first.render();
    await flush();
    first.unmount();

    mocks.setSettingByName.mockClear();
    act(() => abandoned.resolve({ name: "settings/MCP" }));
    await flush();

    expect(mocks.setSettingByName).not.toHaveBeenCalled();
  });
});
