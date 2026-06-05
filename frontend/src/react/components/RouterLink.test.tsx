import type { ReactElement, MouseEvent as ReactMouseEvent } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { RouteTarget } from "@/react/router";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  resolve: vi.fn(() => ({
    href: "/projects/default",
    fullPath: "/projects/default",
  })),
}));

vi.mock("@/react/router", () => ({
  router: {
    push: mocks.push,
    resolve: mocks.resolve,
  },
}));

let RouterLink: typeof import("./RouterLink").RouterLink;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

function getLink(container: HTMLElement): HTMLAnchorElement {
  const link = container.querySelector("a");
  expect(link).toBeInstanceOf(HTMLAnchorElement);
  return link as HTMLAnchorElement;
}

function click(
  link: HTMLAnchorElement,
  options: MouseEventInit = {}
): MouseEvent {
  const event = new MouseEvent("click", {
    bubbles: true,
    cancelable: true,
    button: 0,
    ...options,
  });
  act(() => {
    link.dispatchEvent(event);
  });
  return event;
}

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.resolve.mockReturnValue({
    href: "/projects/default?tab=issues",
    fullPath: "/projects/default?tab=issues",
  });
  ({ RouterLink } = await import("./RouterLink"));
});

describe("RouterLink", () => {
  test("renders resolved href", () => {
    const to: RouteTarget = {
      name: "project.detail",
      params: { projectId: "default" },
    };
    const { container, render, unmount } = renderIntoContainer(
      <RouterLink to={to}>Project</RouterLink>
    );
    render();

    const link = getLink(container);
    expect(mocks.resolve).toHaveBeenCalledWith(to);
    expect(link.getAttribute("href")).toBe("/projects/default?tab=issues");
    expect(link.textContent).toBe("Project");
    unmount();
  });

  test("normal click calls router.push(to) and prevents default", () => {
    const to: RouteTarget = "/projects/default";
    const { container, render, unmount } = renderIntoContainer(
      <RouterLink to={to}>Project</RouterLink>
    );
    render();

    const event = click(getLink(container));

    expect(event.defaultPrevented).toBe(true);
    expect(mocks.push).toHaveBeenCalledWith(to);
    unmount();
  });

  test("meta or ctrl click does not route", () => {
    mocks.resolve.mockReturnValue({
      href: "#project",
      fullPath: "#project",
    });
    const to: RouteTarget = "/projects/default";
    for (const modifier of [{ metaKey: true }, { ctrlKey: true }]) {
      const { container, render, unmount } = renderIntoContainer(
        <RouterLink to={to}>Project</RouterLink>
      );
      render();

      const event = click(getLink(container), modifier);

      expect(event.defaultPrevented).toBe(false);
      unmount();
    }
    expect(mocks.push).not.toHaveBeenCalled();
  });

  test("middle click does not route", () => {
    mocks.resolve.mockReturnValue({
      href: "#project",
      fullPath: "#project",
    });
    const to: RouteTarget = "/projects/default";
    const { container, render, unmount } = renderIntoContainer(
      <RouterLink to={to}>Project</RouterLink>
    );
    render();

    const event = click(getLink(container), { button: 1 });

    expect(event.defaultPrevented).toBe(false);
    expect(mocks.push).not.toHaveBeenCalled();
    unmount();
  });

  test('target="_blank" does not route', () => {
    mocks.resolve.mockReturnValue({
      href: "#project",
      fullPath: "#project",
    });
    const to: RouteTarget = "/projects/default";
    const { container, render, unmount } = renderIntoContainer(
      <RouterLink to={to} target="_blank">
        Project
      </RouterLink>
    );
    render();

    const event = click(getLink(container));

    expect(event.defaultPrevented).toBe(false);
    expect(mocks.push).not.toHaveBeenCalled();
    unmount();
  });

  test("onClick runs before routing", () => {
    const order: string[] = [];
    mocks.push.mockImplementation(() => {
      order.push("push");
    });
    const to: RouteTarget = "/projects/default";
    const { container, render, unmount } = renderIntoContainer(
      <RouterLink
        to={to}
        onClick={() => {
          order.push("onClick");
        }}
      >
        Project
      </RouterLink>
    );
    render();

    click(getLink(container));

    expect(order).toEqual(["onClick", "push"]);
    unmount();
  });

  test("onClick preventDefault skips routing", () => {
    const to: RouteTarget = "/projects/default";
    const { container, render, unmount } = renderIntoContainer(
      <RouterLink
        to={to}
        onClick={(event: ReactMouseEvent<HTMLAnchorElement>) => {
          event.preventDefault();
        }}
      >
        Project
      </RouterLink>
    );
    render();

    const event = click(getLink(container));

    expect(event.defaultPrevented).toBe(true);
    expect(mocks.push).not.toHaveBeenCalled();
    unmount();
  });
});
