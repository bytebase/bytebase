import { render } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { AUTH_SETUP_MODULE, AUTH_SIGNIN_MODULE } from "@/app/router/handles";
import { SplashLayout, SplashPanel } from "./SplashLayout";

const routeState = vi.hoisted(() => ({ routeName: "" }));

// The routed page, which may put a panel of its own in the left column.
vi.mock("react-router", () => ({
  Outlet: () => (
    <div data-testid="auth-page">
      <SplashPanel>
        <p>the page's own panel</p>
      </SplashPanel>
    </div>
  ),
  useMatches: () => [{ handle: { name: routeState.routeName } }],
}));

// jsdom has no matchMedia; `wide` is whether the viewport is at least lg.
function renderLayout(routeName: string, { wide = true } = {}) {
  routeState.routeName = routeName;
  window.matchMedia = ((query: string) => ({
    matches: wide,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => true,
  })) as unknown as typeof window.matchMedia;
  return render(<SplashLayout />);
}

describe("SplashLayout", () => {
  test("renders a page alone in one full-width column, whatever panel the page offers", () => {
    const { container, getByTestId, queryByText } = renderLayout(
      AUTH_SIGNIN_MODULE
    );

    const columns = container.firstElementChild?.children ?? [];
    expect(columns).toHaveLength(1);
    expect(columns[0]).toContainElement(getByTestId("auth-page"));
    expect(columns[0]).not.toHaveClass("lg:w-1/2");
    expect(columns[0]).not.toHaveClass("lg:flex-none");
    expect(queryByText("the page's own panel")).not.toBeInTheDocument();
  });

  test("gives the left column to the setup wizard's panel from lg", () => {
    const { container, getByText, getByTestId } =
      renderLayout(AUTH_SETUP_MODULE);

    const panel = getByText("the page's own panel");
    const [leftColumn, rightColumn] = container.firstElementChild?.children ?? [];
    expect(leftColumn).toContainElement(panel);
    expect(rightColumn).toContainElement(getByTestId("auth-page"));
    expect(rightColumn).not.toContainElement(panel);
    expect(rightColumn).toHaveClass("lg:w-1/2", "lg:flex-none");
  });

  test("gives the setup page no slot below lg, where the left column is hidden", () => {
    const { getByTestId, queryByText } = renderLayout(AUTH_SETUP_MODULE, {
      wide: false,
    });

    expect(getByTestId("auth-page")).toBeInTheDocument();
    expect(queryByText("the page's own panel")).not.toBeInTheDocument();
  });
});
