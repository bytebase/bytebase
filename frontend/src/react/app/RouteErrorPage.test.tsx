import { act } from "react";
import { createRoot } from "react-dom/client";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { RouteErrorPage } from "./RouteErrorPage";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

const Boom = () => {
  throw new Error("boom from render");
};

describe("RouteErrorPage", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => {
      root.unmount();
    });
    document.body.removeChild(container);
  });

  test("inline variant renders a panel without the full-screen container", async () => {
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    try {
      const router = createMemoryRouter(
        [
          {
            path: "/",
            element: <Boom />,
            errorElement: <RouteErrorPage inline />,
          },
        ],
        { initialEntries: ["/"] }
      );
      await act(async () => {
        root.render(<RouterProvider router={router} />);
      });
      expect(container.textContent).toContain(
        "error-page.something-went-wrong"
      );
      // The inline panel must not claim the whole viewport — it renders
      // inside a layout's content area.
      expect(container.querySelector(".h-screen")).toBeNull();
    } finally {
      consoleError.mockRestore();
    }
  });

  test("a pathless errorElement route keeps the parent layout shell alive", async () => {
    // The tier-2 "layout seam" pattern: the boundary lives BELOW the
    // layout route, so a crashing page renders the error panel inside the
    // layout's outlet — navigation chrome survives. (An errorElement ON
    // the layout route would replace the layout itself.)
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    try {
      const Shell = () => (
        <div>
          <nav>APP-SHELL</nav>
          <Outlet />
        </div>
      );
      const router = createMemoryRouter(
        [
          {
            path: "/",
            element: <Shell />,
            children: [
              {
                errorElement: <RouteErrorPage inline />,
                children: [{ index: true, element: <Boom /> }],
              },
            ],
          },
        ],
        { initialEntries: ["/"] }
      );
      await act(async () => {
        root.render(<RouterProvider router={router} />);
      });
      expect(container.textContent).toContain("APP-SHELL");
      expect(container.textContent).toContain(
        "error-page.something-went-wrong"
      );
    } finally {
      consoleError.mockRestore();
    }
  });

  test("a route render crash shows the recovery page, with raw details in dev mode", async () => {
    // Vitest runs with import.meta.env.DEV = true, so this covers the
    // dev/diagnostic contract: raw error details are shown for debugging.
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    try {
      const router = createMemoryRouter(
        [{ path: "/", element: <Boom />, errorElement: <RouteErrorPage /> }],
        { initialEntries: ["/"] }
      );
      await act(async () => {
        root.render(<RouterProvider router={router} />);
      });
      expect(container.textContent).toContain(
        "error-page.something-went-wrong"
      );
      expect(container.textContent).toContain("boom from render");
      // Recovery actions.
      expect(container.textContent).toContain("common.refresh");
      expect(container.textContent).toContain("error-page.go-back-home");
    } finally {
      consoleError.mockRestore();
    }
  });

  test("production hides raw error details and logs them to the console instead", async () => {
    // The recovery page catches render/loader/lazy-route failures across
    // the whole app — raw stacks and loader error text can leak bundle
    // paths, component names, or backend messages. Ordinary users get
    // generic copy + recovery actions; the raw error goes to the console
    // for diagnostics (react-router does not log it for custom
    // errorElements).
    vi.stubEnv("DEV", false);
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    try {
      const router = createMemoryRouter(
        [{ path: "/", element: <Boom />, errorElement: <RouteErrorPage /> }],
        { initialEntries: ["/"] }
      );
      await act(async () => {
        root.render(<RouterProvider router={router} />);
      });
      expect(container.textContent).toContain(
        "error-page.something-went-wrong"
      );
      expect(container.textContent).toContain("common.refresh");
      expect(container.textContent).toContain("error-page.go-back-home");
      // No raw error text or details block for end users.
      expect(container.textContent).not.toContain("boom from render");
      expect(container.querySelector("details")).toBeNull();
      // The raw error is still logged for diagnostics.
      expect(
        consoleError.mock.calls.some(
          (args) =>
            typeof args[0] === "string" && args[0].includes("[RouteErrorPage]")
        )
      ).toBe(true);
    } finally {
      consoleError.mockRestore();
      vi.unstubAllEnvs();
    }
  });
});
