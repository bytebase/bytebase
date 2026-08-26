import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useAuthStore: vi.fn(),
  useAppStore: vi.fn(),
  useWorkspace: vi.fn(),
  isLoggedIn: { value: true },
  isSaaSMode: { value: false },
  currentWorkspace: {
    value: { name: "workspaces/ws-1", title: "Acme Corp" } as
      | {
          name: string;
          title: string;
        }
      | undefined,
  },
  workspaceList: { value: [] as { name: string; title: string }[] },
  loadWorkspace: vi.fn(async () => {}),
  loadWorkspaceList: vi.fn(async () => {}),
  switchWorkspace: vi.fn(async () => {}),
  routerReplace: vi.fn(),
  routerBack: vi.fn(),
  currentRoute: {
    value: {
      query: {} as Record<string, string>,
      fullPath: "/oauth2/consent?x=1",
    },
  },
  fetchImpl: vi.fn(),
  getMCPInfo: vi.fn(),
}));
mocks.useAuthStore.mockImplementation(() => ({
  get isLoggedIn() {
    return mocks.isLoggedIn.value;
  },
}));
mocks.useWorkspace.mockImplementation(() => mocks.currentWorkspace.value);
// The OAuth2 consent page selects discrete app-store slices via
// `useAppStore((state) => state.X)`. Resolve each selector against a
// mock state that exposes the workspace list (live via getter) plus
// the action mocks under test. `isSaaSMode` is now an app-store method.
mocks.useAppStore.mockImplementation((selector: (state: unknown) => unknown) =>
  selector({
    get workspaceList() {
      return mocks.workspaceList.value;
    },
    isSaaSMode: () => mocks.isSaaSMode.value,
    isLoggedIn: () => mocks.isLoggedIn.value,
    loadWorkspace: mocks.loadWorkspace,
    loadWorkspaceList: mocks.loadWorkspaceList,
    switchWorkspace: mocks.switchWorkspace,
  })
);
// The consent page also calls `useAppStore.getState().loadServerInfo()` on mount.
(mocks.useAppStore as unknown as { getState: () => unknown }).getState =
  () => ({
    loadServerInfo: vi.fn().mockResolvedValue(undefined),
  });

vi.mock("@/hooks/useAppState", () => ({
  useWorkspace: mocks.useWorkspace,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: mocks.useAppStore,
}));

vi.mock("@/stores", () => ({
  useAuthStore: mocks.useAuthStore,
}));

// Test-only Select stub: Base UI's Select renders its popup through a portal,
// which makes click-through-portal flows fragile in jsdom. We swap in a native
// <select> here so we can exercise the consent page's switch wiring directly.
// The real Select component is covered by its own tests.
vi.mock("@/components/ui/select", () => ({
  Select: ({
    value,
    onValueChange,
    children,
    disabled,
  }: {
    value: string;
    onValueChange?: (v: string) => void;
    children: ReactNode;
    disabled?: boolean;
  }) => (
    <select
      data-testid="workspace-select"
      value={value}
      disabled={disabled}
      onChange={(e) => onValueChange?.(e.target.value)}
    >
      {children}
    </select>
  ),
  SelectTrigger: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  SelectValue: () => null,
  SelectContent: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  SelectItem: ({ value, children }: { value: string; children: ReactNode }) => (
    <option value={value}>{children}</option>
  ),
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    replace: mocks.routerReplace,
    back: mocks.routerBack,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/components/BytebaseLogo", () => ({
  BytebaseLogo: () => null,
}));

vi.mock("@/api", () => ({
  workspaceServiceClientConnect: { getMCPInfo: mocks.getMCPInfo },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, string>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
  // The page now reaches the connect client for GetMCPInfo, and @/api's error
  // middleware imports the shared i18n instance, which registers this.
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

let OAuth2ConsentPage: typeof import("./OAuth2ConsentPage").OAuth2ConsentPage;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

// Every real GetMCPInfo response carries one row per ceiling the gate serves.
// The page reads that table to tell a policy it can disclose from one the
// server refuses, so a fixture without it is not a response the server sends.
const SERVED_MODES = [
  { capability: 1 },
  { capability: 3 },
  { capability: 4 },
];

const flushPromises = () =>
  act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.isLoggedIn.value = true;
  mocks.isSaaSMode.value = false;
  mocks.currentWorkspace.value = {
    name: "workspaces/ws-1",
    title: "Acme Corp",
  };
  mocks.workspaceList.value = [];
  mocks.currentRoute.value.query = {};
  mocks.currentRoute.value.fullPath = "/oauth2/consent";
  globalThis.fetch = mocks.fetchImpl as typeof fetch;
  mocks.fetchImpl.mockReset();
  mocks.getMCPInfo.mockReset();
  // Default: a served read-only ceiling, the page's ordinary case. Allow
  // renders only under one, so a failing default would leave every test that
  // is not about the ceiling asserting against the undisclosed card.
  mocks.getMCPInfo.mockResolvedValue({
    capability: 3,
    ignoreMaskingExemptions: false,
    dataMaskingAvailable: true,
    modes: SERVED_MODES,
    methods: [],
    engines: [],
  });
  ({ OAuth2ConsentPage } = await import("./OAuth2ConsentPage"));
});

describe("OAuth2ConsentPage", () => {
  test("redirects to signin when user is not logged in", async () => {
    mocks.isLoggedIn.value = false;
    mocks.currentRoute.value.fullPath = "/oauth2/consent?client_id=abc";
    const { render, unmount } = renderIntoContainer(<OAuth2ConsentPage />);
    render();
    await flushPromises();
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "auth.signin",
      query: { redirect: "/oauth2/consent?client_id=abc" },
    });
    unmount();
  });

  test("renders error when required params are missing", async () => {
    mocks.currentRoute.value.query = { client_id: "abc" };
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    // The component now resolves error text via the i18n key. Our t() mock
    // returns the key as-is, so we assert on the key rather than English.
    expect(container.textContent).toContain(
      "oauth2.consent.error-missing-params"
    );
    unmount();
  });

  test("fetches client info and renders consent form", async () => {
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    expect(mocks.fetchImpl).toHaveBeenCalledWith("/api/oauth2/clients/c1");
    expect(container.textContent).toContain("Acme");
    expect(container.querySelector('form[method="POST"]')).not.toBeNull();
    const hiddenClientId = container.querySelector<HTMLInputElement>(
      'input[name="client_id"]'
    );
    expect(hiddenClientId?.value).toBe("c1");
    unmount();
  });

  test("renders error when client lookup fails", async () => {
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: false,
      json: async () => ({ error_description: "client unknown" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    expect(container.textContent).toContain("client unknown");
    unmount();
  });

  test("shows current workspace title on the consent card", async () => {
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    expect(container.textContent).toContain("oauth2.consent.workspace-label");
    expect(container.textContent).toContain("Acme Corp");
    // Self-hosted (default in this test) does NOT prefetch the workspace list.
    expect(mocks.loadWorkspaceList).not.toHaveBeenCalled();
    unmount();
  });

  test("prefetches workspace list and shows picker on SaaS with multiple workspaces", async () => {
    mocks.isSaaSMode.value = true;
    mocks.workspaceList.value = [
      { name: "workspaces/ws-1", title: "Acme Corp" },
      { name: "workspaces/ws-2", title: "Side Project" },
    ];
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    expect(mocks.loadWorkspaceList).toHaveBeenCalledTimes(1);
    // Picker trigger renders the current workspace title.
    expect(container.textContent).toContain("Acme Corp");
    unmount();
  });

  test("picking a different workspace calls SwitchWorkspace and reloads", async () => {
    mocks.isSaaSMode.value = true;
    mocks.workspaceList.value = [
      { name: "workspaces/ws-1", title: "Acme Corp" },
      { name: "workspaces/ws-2", title: "Side Project" },
    ];
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });

    // Stub globalThis.location.reload so the test doesn't actually navigate.
    // window === globalThis in jsdom, so this also stubs the value the
    // component reads via globalThis.location.reload().
    const reload = vi.fn();
    Object.defineProperty(globalThis, "location", {
      writable: true,
      value: { ...globalThis.location, reload },
    });

    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();

    const select = container.querySelector<HTMLSelectElement>(
      'select[data-testid="workspace-select"]'
    );
    expect(select).not.toBeNull();
    expect(select?.value).toBe("workspaces/ws-1");

    await act(async () => {
      select!.value = "workspaces/ws-2";
      select!.dispatchEvent(new Event("change", { bubbles: true }));
      await Promise.resolve();
    });
    await flushPromises();

    // Verifies the consent page calls the workspace store's
    // *withoutRedirect* variant, which posts on the store's own channel —
    // crucially, that variant does NOT fire the store's onmessage handler
    // in this tab, so we don't race-redirect to the landing page and lose
    // the OAuth query params.
    expect(mocks.switchWorkspace).toHaveBeenCalledTimes(1);
    expect(mocks.switchWorkspace).toHaveBeenCalledWith(
      "workspaces/ws-2",
      false
    );
    expect(reload).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("picking the same workspace is a no-op", async () => {
    mocks.isSaaSMode.value = true;
    mocks.workspaceList.value = [
      { name: "workspaces/ws-1", title: "Acme Corp" },
      { name: "workspaces/ws-2", title: "Side Project" },
    ];
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();

    const select = container.querySelector<HTMLSelectElement>(
      'select[data-testid="workspace-select"]'
    );
    await act(async () => {
      // Re-dispatching the same value should not trigger a switch.
      select!.dispatchEvent(new Event("change", { bubbles: true }));
      await Promise.resolve();
    });
    expect(mocks.switchWorkspace).not.toHaveBeenCalled();
    unmount();
  });

  test("deny creates a programmatic form and submits it", async () => {
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    const denyBtn = Array.from(
      container.querySelectorAll<HTMLButtonElement>("button")
    ).find((b) => b.textContent === "common.deny");
    expect(denyBtn).toBeDefined();
    const submitSpy = vi
      .spyOn(HTMLFormElement.prototype, "submit")
      .mockImplementation(() => {});
    act(() => {
      denyBtn?.click();
    });
    expect(submitSpy).toHaveBeenCalledTimes(1);
    submitSpy.mockRestore();
    unmount();
  });

  // The consent page is the only hop between /authorize's validation of
  // `resource`/`scope` and the POST that persists them. Dropping either one
  // silently unbinds the grant, and the allow and deny paths build their field
  // lists separately, so both are asserted.
  test("forwards resource and scope on both the allow and deny paths", async () => {
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
      resource: "https://bb.example.com/mcp",
      scope: "mcp:read-only",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();

    const allowForm = container.querySelector<HTMLFormElement>(
      'form[method="POST"]'
    );
    expect(
      allowForm?.querySelector<HTMLInputElement>('input[name="resource"]')?.value
    ).toBe("https://bb.example.com/mcp");
    expect(
      allowForm?.querySelector<HTMLInputElement>('input[name="scope"]')?.value
    ).toBe("mcp:read-only");

    const denyBtn = Array.from(
      container.querySelectorAll<HTMLButtonElement>("button")
    ).find((b) => b.textContent === "common.deny");
    let denyFields: Record<string, string> = {};
    const submitSpy = vi
      .spyOn(HTMLFormElement.prototype, "submit")
      .mockImplementation(function (this: HTMLFormElement) {
        denyFields = Object.fromEntries(
          Array.from(this.querySelectorAll("input")).map((i) => [
            i.name,
            i.value,
          ])
        );
      });
    act(() => {
      denyBtn?.click();
    });
    expect(denyFields.resource).toBe("https://bb.example.com/mcp");
    expect(denyFields.scope).toBe("mcp:read-only");
    submitSpy.mockRestore();
    unmount();
  });

  // The ceiling states. The same ceiling refuses the POST server-side, so what
  // these pin is that the page says the same thing first — including the one
  // state where there is nothing to approve.
  const consentQuery = () => ({
    client_id: "c1",
    redirect_uri: "https://app/callback",
    state: "s",
    code_challenge: "ch",
    code_challenge_method: "S256",
  });

  const renderWithCeiling = async (info: Record<string, unknown>) => {
    mocks.currentRoute.value.query = consentQuery();
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    mocks.getMCPInfo.mockResolvedValue({ modes: SERVED_MODES, ...info });
    const handle = renderIntoContainer(<OAuth2ConsentPage />);
    handle.render();
    await flushPromises();
    return handle;
  };

  // Codex, #21237: the !response.ok branch returned, its sibling catch did not.
  // The render checks loading before error, so a failed client lookup sat
  // behind the spinner until an optional policy request it no longer needed
  // finally settled.
  test("a failed client lookup shows its error without waiting on the policy", async () => {
    mocks.currentRoute.value.query = consentQuery();
    mocks.fetchImpl.mockRejectedValue(new Error("network down"));
    // Never settles: if the error waits on this, the test sees the spinner.
    mocks.getMCPInfo.mockReturnValue(new Promise(() => {}));

    const handle = renderIntoContainer(<OAuth2ConsentPage />);
    handle.render();
    await flushPromises();

    expect(handle.container.textContent).toContain(
      "oauth2.consent.error-load-failed"
    );
    expect(handle.container.querySelector(".animate-spin")).toBeNull();
    handle.unmount();
  });

  test("a read-only ceiling says what the session may not do", async () => {
    const { container, unmount } = await renderWithCeiling({
      capability: 3,
      ignoreMaskingExemptions: false,
      methods: [],
      engines: [],
    });
    expect(container.textContent).toContain("oauth2.consent.mcp.line.read");
    expect(container.textContent).toContain("oauth2.consent.mcp.line.no-write");
    expect(container.textContent).not.toContain("oauth2.consent.mcp.line.write");
    // The masking line is the toggle's, not the ceiling's.
    expect(container.textContent).not.toContain(
      "oauth2.consent.mcp.line.masking"
    );
    expect(container.textContent).toContain("common.allow");
    unmount();
  });

  test("a read-write ceiling adds the write line and the caution", async () => {
    const { container, unmount } = await renderWithCeiling({
      capability: 4,
      ignoreMaskingExemptions: true,
      dataMaskingAvailable: true,
      methods: [],
      engines: [],
    });
    expect(container.textContent).toContain("oauth2.consent.mcp.line.write");
    expect(container.textContent).toContain("oauth2.consent.mcp.write-caution");
    expect(container.textContent).toContain("oauth2.consent.mcp.line.masking");
    unmount();
  });

  // The masking line promises a restriction. The toggle withholds unmasking
  // exemptions from MCP sessions, which restricts nothing on a workspace where
  // masking does not run — and this card is read at the moment someone decides
  // whether to hand over access.
  test("the masking line is not promised where masking does not run", async () => {
    const { container, unmount } = await renderWithCeiling({
      capability: 4,
      ignoreMaskingExemptions: true,
      dataMaskingAvailable: false,
      methods: [],
      engines: [],
    });
    // The rest of the card is unchanged, so this is the line and not the card.
    expect(container.textContent).toContain("oauth2.consent.mcp.line.write");
    expect(container.textContent).not.toContain(
      "oauth2.consent.mcp.line.masking"
    );
    unmount();
  });

  // Codex, #21237: the disabled screen's only action was router.back(), so the
  // OAuth client sat waiting on a callback that never came. A deny POST returns
  // access_denied to the registered redirect_uri, which is the answer it is
  // blocked on.
  test("dismissing a disabled ceiling denies the request instead of going back", async () => {
    const { container, unmount } = await renderWithCeiling({
      capability: 1,
      ignoreMaskingExemptions: false,
      dataMaskingAvailable: true,
      methods: [],
      engines: [],
    });

    const submitted: HTMLFormElement[] = [];
    const realSubmit = HTMLFormElement.prototype.submit;
    HTMLFormElement.prototype.submit = function submit(this: HTMLFormElement) {
      submitted.push(this);
    };
    try {
      const close = [...container.querySelectorAll("button")].find((b) =>
        b.textContent?.includes("common.close")
      );
      act(() => (close as HTMLButtonElement)?.click());
    } finally {
      HTMLFormElement.prototype.submit = realSubmit;
    }

    expect(mocks.routerBack).not.toHaveBeenCalled();
    expect(submitted).toHaveLength(1);
    const action = submitted[0].querySelector('input[name="action"]');
    expect((action as HTMLInputElement)?.value).toBe("deny");
    unmount();
  });

  // Codex, #21237: a SaaS user whose current workspace has MCP off could not
  // switch to one that permits it without abandoning the OAuth flow.
  test("the disabled screen keeps the workspace switcher", async () => {
    const { container, unmount } = await renderWithCeiling({
      capability: 1,
      ignoreMaskingExemptions: false,
      dataMaskingAvailable: true,
      methods: [],
      engines: [],
    });
    expect(container.textContent).toContain("oauth2.consent.workspace-label");
    unmount();
  });

  test("a disabled ceiling offers nothing to approve", async () => {
    const { container, unmount } = await renderWithCeiling({
      capability: 1,
      ignoreMaskingExemptions: false,
      methods: [],
      engines: [],
    });
    expect(container.textContent).toContain("oauth2.consent.mcp.disabled.title");
    expect(container.textContent).toContain(
      "oauth2.consent.mcp.disabled.ask-admin"
    );
    // Nothing to approve and nothing to deny: the grant is not on offer.
    expect(container.textContent).not.toContain("common.allow");
    expect(container.textContent).not.toContain("common.deny");
    unmount();
  });

  // BOT-106, and the reason this is a blocker rather than a tidy-up. The page
  // used to fall through to a generic "access your account" card with Allow
  // live whenever GetMCPInfo failed for ANY reason. The two broken-ceiling
  // cases were cosmetic — the POST refuses those anyway — but a transient
  // failure is not: the POST reads the ceiling for itself, succeeds, and issues
  // a grant against a card that never named the ceiling it was granting.
  //
  // A timeout arrives here the same way: the client throws, and the page cannot
  // tell a deadline from a refusal, which is the whole point of failing closed.
  test("a failed policy read offers no grant and can be retried", async () => {
    mocks.currentRoute.value.query = consentQuery();
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    mocks.getMCPInfo.mockRejectedValueOnce(new Error("deadline exceeded"));

    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();

    expect(container.textContent).toContain(
      "oauth2.consent.mcp.undisclosed.unknown.title"
    );
    // The hole: neither the grant button nor the form that carries it.
    expect(container.textContent).not.toContain("common.allow");
    expect(container.querySelector('form[method="POST"]')).toBeNull();

    // The default mock resolves, so the retry reaches a ceiling and the page
    // becomes the ordinary consent card — the failure was not terminal.
    const retry = [...container.querySelectorAll("button")].find((b) =>
      b.textContent?.includes("oauth2.consent.mcp.undisclosed.unknown.retry")
    );
    expect(retry).toBeDefined();
    await act(async () => {
      retry?.click();
    });
    await flushPromises();
    expect(container.textContent).toContain("oauth2.consent.mcp.line.read");
    expect(container.textContent).toContain("common.allow");
    unmount();
  });

  test("an unreadable ceiling says so and offers no grant", async () => {
    const { container, unmount } = await renderWithCeiling({
      capability: 0,
      policyUnreadable: true,
      ignoreMaskingExemptions: false,
      dataMaskingAvailable: true,
      methods: [],
      engines: [],
    });
    expect(container.textContent).toContain(
      "oauth2.consent.mcp.undisclosed.unreadable.title"
    );
    expect(container.textContent).not.toContain("common.allow");
    // Nothing an admin has not fixed will change this answer, so there is no
    // button implying a retry might. The label is built from the reason, so
    // this is the only key a retry here could render — naming any other
    // reason's key would assert something this render cannot produce.
    expect(container.textContent).not.toContain(
      "oauth2.consent.mcp.undisclosed.unreadable.retry"
    );
    // The workspace row stays, so a SaaS user can switch to one that discloses.
    expect(container.textContent).toContain("oauth2.consent.workspace-label");
    // Codex, #21254: the line that explains why there is no Allow arrives after
    // the policy read settles, so it has to be announced rather than merely
    // styled. A plain div renders identically and says nothing.
    const status = container.querySelector('[role="alert"]');
    expect(status).not.toBeNull();
    expect(status?.textContent).toContain(
      "oauth2.consent.mcp.undisclosed.unreadable.line"
    );
    unmount();
  });

  test("a ceiling no mode serves says so and offers no grant", async () => {
    const { container, unmount } = await renderWithCeiling({
      // The reserved 2, or a ceiling a newer release wrote. It parses, so only
      // its absence from the serving table catches it.
      capability: 2,
      ignoreMaskingExemptions: false,
      dataMaskingAvailable: true,
      methods: [],
      engines: [],
    });
    expect(container.textContent).toContain(
      "oauth2.consent.mcp.undisclosed.unserved.title"
    );
    expect(container.textContent).not.toContain("common.allow");
    // The other state only an admin can fix, held to the same rule.
    expect(container.textContent).not.toContain(
      "oauth2.consent.mcp.undisclosed.unserved.retry"
    );
    unmount();
  });

  // The server serves this ceiling and this bundle has no word for it: a newer
  // release's value against a page that was already open. Falling through to
  // the read-only wording would understate what was approved, and telling the
  // user to find an admin would send them after a policy that is working.
  test("a ceiling this page cannot name asks for a reload, not a grant", async () => {
    const { container, unmount } = await renderWithCeiling({
      capability: 5,
      ignoreMaskingExemptions: false,
      dataMaskingAvailable: true,
      modes: [...SERVED_MODES, { capability: 5 }],
      methods: [],
      engines: [],
    });
    expect(container.textContent).toContain(
      "oauth2.consent.mcp.undisclosed.outdated.title"
    );
    expect(container.textContent).not.toContain("common.allow");

    const reload = vi.fn();
    Object.defineProperty(globalThis, "location", {
      writable: true,
      value: { ...globalThis.location, reload },
    });
    const retry = [...container.querySelectorAll("button")].find((b) =>
      b.textContent?.includes("oauth2.consent.mcp.undisclosed.outdated.retry")
    );
    expect(retry).toBeDefined();
    await act(async () => {
      retry?.click();
    });
    await flushPromises();
    // A reload, not a re-read: only a fresh bundle can name this ceiling, so
    // refetching the policy would return the same value with no word for it.
    expect(reload).toHaveBeenCalledTimes(1);
    expect(mocks.getMCPInfo).toHaveBeenCalledTimes(1);
    unmount();
  });

  // Same reasoning as the disabled screen: history leaves the OAuth client
  // waiting on a callback that never comes, and these four states are the ones
  // where the person has nothing else to do here.
  test("dismissing an undisclosed policy denies the request", async () => {
    const { container, unmount } = await renderWithCeiling({
      capability: 0,
      policyUnreadable: true,
      ignoreMaskingExemptions: false,
      dataMaskingAvailable: true,
      methods: [],
      engines: [],
    });

    const submitted: HTMLFormElement[] = [];
    const realSubmit = HTMLFormElement.prototype.submit;
    HTMLFormElement.prototype.submit = function submit(this: HTMLFormElement) {
      submitted.push(this);
    };
    try {
      const close = [...container.querySelectorAll("button")].find((b) =>
        b.textContent?.includes("common.close")
      );
      act(() => (close as HTMLButtonElement)?.click());
    } finally {
      HTMLFormElement.prototype.submit = realSubmit;
    }

    expect(mocks.routerBack).not.toHaveBeenCalled();
    expect(submitted).toHaveLength(1);
    const action = submitted[0].querySelector('input[name="action"]');
    expect((action as HTMLInputElement)?.value).toBe("deny");
    unmount();
  });
});
