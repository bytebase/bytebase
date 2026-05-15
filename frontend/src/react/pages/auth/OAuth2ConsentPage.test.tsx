import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>((getter) => getter()),
  useAuthStore: vi.fn(),
  useActuatorV1Store: vi.fn(),
  useWorkspaceV1Store: vi.fn(),
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
  fetchWorkspaceList: vi.fn(async () => {}),
  switchWorkspaceWithoutRedirect: vi.fn(async () => {}),
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
mocks.useActuatorV1Store.mockImplementation(() => ({
  get isSaaSMode() {
    return mocks.isSaaSMode.value;
  },
}));
mocks.useWorkspaceV1Store.mockImplementation(() => ({
  get currentWorkspace() {
    return mocks.currentWorkspace.value;
  },
  get workspaceList() {
    return mocks.workspaceList.value;
  },
  fetchWorkspaceList: mocks.fetchWorkspaceList,
  switchWorkspaceWithoutRedirect: mocks.switchWorkspaceWithoutRedirect,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useAuthStore: mocks.useAuthStore,
  useActuatorV1Store: mocks.useActuatorV1Store,
  useWorkspaceV1Store: mocks.useWorkspaceV1Store,
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

vi.mock("@/router", () => ({
  router: {
    replace: mocks.routerReplace,
    back: mocks.routerBack,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/router/auth", () => ({
  AUTH_SIGNIN_MODULE: "auth.signin",
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
    expect(mocks.fetchWorkspaceList).not.toHaveBeenCalled();
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
    expect(mocks.fetchWorkspaceList).toHaveBeenCalledTimes(1);
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
    expect(mocks.switchWorkspaceWithoutRedirect).toHaveBeenCalledTimes(1);
    expect(mocks.switchWorkspaceWithoutRedirect).toHaveBeenCalledWith(
      "workspaces/ws-2"
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
    expect(mocks.switchWorkspaceWithoutRedirect).not.toHaveBeenCalled();
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
