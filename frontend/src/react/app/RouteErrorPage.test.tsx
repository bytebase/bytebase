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

  test("a route render crash shows the recovery page with error details", async () => {
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
      // The thrown error is surfaced so users can report it.
      expect(container.textContent).toContain("boom from render");
      // Recovery actions.
      expect(container.textContent).toContain("common.refresh");
      expect(container.textContent).toContain("error-page.go-back-home");
    } finally {
      consoleError.mockRestore();
    }
  });
});
