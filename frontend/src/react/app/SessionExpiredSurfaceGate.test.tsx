import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { ref } from "vue";

const unauthenticatedOccurredRef = ref(false);
const isLoggedInRef = ref(true);
const fullPathRef = ref("/");
const routeNameRef = ref("workspace.dashboard");

vi.mock("@/react/hooks/useVueState", () => ({
  // Passthrough — calls getter() synchronously on each render. Does NOT
  // re-render when Vue state changes after mount; set state on the refs
  // above BEFORE rendering in each test.
  useVueState: <T,>(getter: () => T) => getter(),
}));

vi.mock("@/store", () => ({
  useAuthStore: () => ({
    get unauthenticatedOccurred() {
      return unauthenticatedOccurredRef.value;
    },
    get isLoggedIn() {
      return isLoggedInRef.value;
    },
  }),
}));

vi.mock("@/router", () => ({
  router: {
    currentRoute: {
      get value() {
        return { fullPath: fullPathRef.value, name: routeNameRef.value };
      },
    },
  },
}));

vi.mock("@/utils/auth", () => ({
  isAuthRelatedRoute: (name: string) =>
    [
      "auth.signin",
      "auth.signin.admin",
      "auth.signup",
      "auth.mfa",
      "auth.password.reset",
      "auth.password.forgot",
      "auth.oauth.callback",
      "auth.oidc.callback",
    ].includes(name),
}));

vi.mock("@/react/components/auth/SessionExpiredSurface", () => ({
  SessionExpiredSurface: ({ currentPath }: { currentPath: string }) => (
    <div data-testid="surface" data-path={currentPath} />
  ),
}));

import { SessionExpiredSurfaceGate } from "./SessionExpiredSurfaceGate";

describe("SessionExpiredSurfaceGate", () => {
  let container: HTMLDivElement;
  let root: Root;

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    unauthenticatedOccurredRef.value = false;
    isLoggedInRef.value = true;
    fullPathRef.value = "/";
    routeNameRef.value = "workspace.dashboard";
  });

  const render = () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    act(() => root.render(<SessionExpiredSurfaceGate />));
  };

  test("renders nothing when unauthenticatedOccurred is false", () => {
    render();
    expect(container.querySelector("[data-testid='surface']")).toBeNull();
  });

  test("renders SessionExpiredSurface with current path when triggered", () => {
    unauthenticatedOccurredRef.value = true;
    fullPathRef.value = "/projects/sample/plans/123";
    render();
    const surface = container.querySelector("[data-testid='surface']");
    expect(surface).not.toBeNull();
    expect(surface?.getAttribute("data-path")).toBe(
      "/projects/sample/plans/123"
    );
  });

  test("renders nothing while on an auth-related route", () => {
    unauthenticatedOccurredRef.value = true;
    routeNameRef.value = "auth.signin";
    render();
    expect(container.querySelector("[data-testid='surface']")).toBeNull();
  });

  test("renders nothing when user is not logged in", () => {
    unauthenticatedOccurredRef.value = true;
    isLoggedInRef.value = false;
    render();
    expect(container.querySelector("[data-testid='surface']")).toBeNull();
  });
});
