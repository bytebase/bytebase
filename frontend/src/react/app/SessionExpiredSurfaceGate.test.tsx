import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { ref } from "vue";

const unauthenticatedOccurredRef = ref(false);
const fullPathRef = ref("/");

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
  }),
}));

vi.mock("@/router", () => ({
  router: {
    currentRoute: {
      get value() {
        return { fullPath: fullPathRef.value };
      },
    },
  },
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
    fullPathRef.value = "/";
  });

  test("renders nothing when unauthenticatedOccurred is false", () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    act(() => root.render(<SessionExpiredSurfaceGate />));
    expect(container.querySelector("[data-testid='surface']")).toBeNull();
  });

  test("renders SessionExpiredSurface with current path when true", () => {
    unauthenticatedOccurredRef.value = true;
    fullPathRef.value = "/projects/sample/plans/123";
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    act(() => root.render(<SessionExpiredSurfaceGate />));
    const surface = container.querySelector("[data-testid='surface']");
    expect(surface).not.toBeNull();
    expect(surface?.getAttribute("data-path")).toBe(
      "/projects/sample/plans/123"
    );
  });
});
