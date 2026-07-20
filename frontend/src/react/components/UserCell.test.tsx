import type {
  ButtonHTMLAttributes,
  MouseEvent as ReactMouseEvent,
  ReactElement,
  ReactNode,
} from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("./RouterLink", () => ({
  RouterLink: ({
    children,
    className,
    onClick,
    to,
  }: {
    children?: ReactNode;
    className?: string;
    onClick?: (event: ReactMouseEvent<HTMLAnchorElement>) => void;
    to: { name?: string };
  }) =>
    createElement(
      "a",
      {
        className,
        "data-route-name": to.name,
        href: "#",
        onClick,
      },
      children
    ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    ...props
  }: ButtonHTMLAttributes<HTMLButtonElement>) =>
    createElement(
      "button",
      {
        ...props,
        "data-ui-button": "true",
        onClick,
      },
      children
    ),
}));

let UserCell: typeof import("./UserCell").UserCell;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  ({ UserCell } = await import("./UserCell"));
});

describe("UserCell", () => {
  test("uses the shared Button for callback name links", () => {
    const onNameClick = vi.fn();
    const onRowClick = vi.fn();
    const { container, unmount } = renderIntoContainer(
      <div onClick={onRowClick}>
        <UserCell
          title="Dev User"
          subtitle="dev@example.com"
          nameLink={{ onClick: onNameClick }}
        />
      </div>
    );

    const button = container.querySelector("button");
    expect(button?.getAttribute("data-ui-button")).toBe("true");

    act(() => {
      button?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(onNameClick).toHaveBeenCalledTimes(1);
    expect(onRowClick).not.toHaveBeenCalled();
    unmount();
  });

  test("renders route name links without bubbling to row click handlers", () => {
    const onRowClick = vi.fn();
    const { container, unmount } = renderIntoContainer(
      <div onClick={onRowClick}>
        <UserCell
          title="Dev User"
          subtitle="dev@example.com"
          nameLink={{
            to: {
              name: "workspace.user-profile",
              params: { principalEmail: "dev@example.com" },
            },
          }}
        />
      </div>
    );

    const link = container.querySelector("a");
    expect(link).toBeInstanceOf(HTMLAnchorElement);
    expect(link?.getAttribute("data-route-name")).toBe(
      "workspace.user-profile"
    );
    expect(link?.textContent).toBe("Dev User");

    act(() => {
      link?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(onRowClick).not.toHaveBeenCalled();
    unmount();
  });
});
