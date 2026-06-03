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

vi.mock("@/react/hooks/useAppState", () => ({
  useWorkspace: mocks.useWorkspace,
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: mocks.useAppStore,
}));

vi.mock("@/store", () => ({
  useAuthStore: mocks.useAuthStore,
}));

// Test-only Select stub: Base UI's Select renders its popup through a portal,
// which makes click-through-portal flows fragile in jsdom. We swap in a native
// <select> here so we can exercise the consent page's switch wiring directly.
// The real Select component is covered by its own tests.
vi.mock("@/react/components/ui/select", () => ({
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

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    replace: mocks.routerReplace,
    back: mocks.routerBack,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/react/components/BytebaseLogo", () => ({
  BytebaseLogo: () => null,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, string>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
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
});
