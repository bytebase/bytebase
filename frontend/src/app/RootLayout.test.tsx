import { act, useEffect, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { createMemoryRouter } from "react-router";
import { RouterProvider } from "react-router/dom";
import { afterEach, describe, expect, test, vi } from "vitest";

const lifecycle = vi.hoisted(() => ({ mounts: 0, unmounts: 0 }));

vi.mock("@/modules/agent/components/AgentWindow", () => ({
  AgentWindow: () => {
    useEffect(() => {
      lifecycle.mounts += 1;
      return () => {
        lifecycle.unmounts += 1;
      };
    }, []);
    return <div data-testid="agent-owner" />;
  },
}));

vi.mock("@/app/AuthGate", () => ({
  AuthGate: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/app/router/NavigationScrollRestoration", () => ({
  NavigationScrollRestoration: ({ children }: { children: ReactNode }) =>
    children,
}));
vi.mock("@/app/router", () => ({
  buildReactRoute: vi.fn(),
  runBeforeEachGuards: () => false,
}));
vi.mock("@/app/router/routes", () => ({ routes: [] }));
vi.mock("@/app/analytics/RouteBehaviorRecorder", () => ({
  RouteBehaviorRecorder: () => null,
}));
vi.mock("@/app/SessionExpiredSurfaceGate", () => ({
  SessionExpiredSurfaceGate: () => <div data-testid="critical-session-gate" />,
}));
vi.mock("@/components/ui/toaster", () => ({ Toaster: () => null }));
vi.mock("@/components/Watermark", () => ({ Watermark: () => null }));

import { RootLayout } from "./RootLayout";

describe("RootLayout", () => {
  let container: HTMLDivElement;
  let root: Root;
  let router: ReturnType<typeof createMemoryRouter>;

  afterEach(() => {
    act(() => root?.unmount());
    router?.dispose();
    container?.remove();
    lifecycle.mounts = 0;
    lifecycle.unmounts = 0;
  });

  test("retains one root Agent owner across SQL Editor and console navigation", async () => {
    router = createMemoryRouter(
      [
        {
          element: <RootLayout />,
          children: [
            { path: "/sql-editor", element: <div>SQL Editor</div> },
            { path: "/projects/example", element: <div>Project</div> },
          ],
        },
      ],
      { initialEntries: ["/sql-editor"] }
    );
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    await act(async () => root.render(<RouterProvider router={router} />));
    const owner = container.querySelector("[data-testid='agent-owner']");

    await act(async () => {
      await router.navigate("/projects/example");
    });
    expect(container.textContent).toContain("Project");
    expect(container.querySelector("[data-testid='agent-owner']")).toBe(owner);
    await act(async () => {
      await router.navigate("/sql-editor");
    });

    expect(container.textContent).toContain("SQL Editor");
    expect(container.querySelectorAll("[data-testid='agent-owner']")).toHaveLength(1);
    expect(container.querySelector("[data-testid='agent-owner']")).toBe(owner);
    expect(lifecycle).toEqual({ mounts: 1, unmounts: 0 });
    expect(container.querySelector("[data-testid='critical-session-gate']")).not.toBeNull();
  });
});
