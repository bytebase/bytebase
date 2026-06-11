import { act } from "react";
import { createRoot } from "react-dom/client";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
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
