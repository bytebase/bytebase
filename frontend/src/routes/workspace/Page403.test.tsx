import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { WORKSPACE_ROUTE_LANDING } from "@/app/router/handles";
import { Page403 } from "./Page403";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  routerPush: vi.fn(),
  currentRoute: {
    value: {
      query: {},
    },
  },
}));

vi.mock("@/app/router", () => ({
  router: {
    currentRoute: mocks.currentRoute,
    push: mocks.routerPush,
    resolve: (to: unknown) => ({ href: String(to), fullPath: String(to) }),
  },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  mocks.routerPush.mockReset();
  mocks.currentRoute.value = { query: {} };
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

describe("Page403", () => {
  test("allows long permission details to wrap within the alert", () => {
    const from = `/403?from=${"segment".repeat(40)}`;
    mocks.currentRoute.value = {
      query: {
        api: "/bytebase.v1.WorkspaceService/GetIamPolicy",
        from,
      },
    };

    act(() => {
      root.render(<Page403 />);
    });

    const details = container.querySelector(".wrap-anywhere");
    expect(details).toBeTruthy();
    expect(details?.className ?? "").toContain("min-w-0");
    expect(details?.textContent).toContain(from);
  });

  test("go back home navigates to the landing page", () => {
    act(() => {
      root.render(<Page403 />);
    });

    const link = Array.from(container.querySelectorAll("a")).find((el) =>
      el.textContent?.includes("error-page.go-back-home")
    );
    expect(link).toBeTruthy();

    act(() => {
      link?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: WORKSPACE_ROUTE_LANDING,
    });
  });
});
