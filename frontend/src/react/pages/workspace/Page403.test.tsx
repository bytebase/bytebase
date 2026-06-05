import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { WORKSPACE_ROUTE_LANDING } from "@/react/router/handles";
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

vi.mock("@/react/router", () => ({
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
