import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";

const unauthenticatedOccurredRef = { value: false };
const isLoggedInRef = { value: true };
const fullPathRef = { value: "/" };
const routeNameRef = { value: "workspace.dashboard" };

vi.mock("@/react/stores/app", () => ({
  useAppStore: <T,>(selector: (s: unknown) => T) =>
    selector({
      isLoggedIn: () => isLoggedInRef.value,
      unauthenticatedOccurred: unauthenticatedOccurredRef.value,
    }),
}));

vi.mock("@/react/router", () => ({
  useCurrentRoute: () => ({
    name: routeNameRef.value,
    fullPath: fullPathRef.value,
    hash: "",
    params: {},
    query: {},
    requiredPermissions: [],
    overrideDocumentTitle: false,
    meta: {},
  }),
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
