import * as stylex from "@stylexjs/stylex";
import { act, type CSSProperties, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Select, SelectContent, SelectItem, SelectTrigger } from "./select";
import { menuRowStateClassName, menuRowStyle } from "./styles.stylex";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const selectMocks = vi.hoisted(() => ({
  positionerProps: [] as Array<Record<string, unknown>>,
}));

vi.mock("@base-ui/react/select", () => ({
  Select: {
    Root: ({ children }: { children: ReactNode }) => <div>{children}</div>,
    Trigger: ({
      children,
      ...props
    }: {
      children: ReactNode;
      className?: string;
    }) => (
      <button type="button" {...props}>
        {children}
      </button>
    ),
    Icon: ({ children }: { children: ReactNode }) => <>{children}</>,
    Value: ({ children }: { children: ReactNode }) => <>{children}</>,
    Portal: ({ children }: { children: ReactNode }) => <>{children}</>,
    Positioner: ({ children, ...props }: { children: ReactNode }) => {
      selectMocks.positionerProps.push(props);
      return <div>{children}</div>;
    },
    Popup: ({ children }: { children: ReactNode }) => <div>{children}</div>,
    Item: ({
      children,
      ...props
    }: {
      children: ReactNode;
      className?: string;
      role?: string;
      style?: CSSProperties;
    }) => (
      <div role="option" {...props}>
        {children}
      </div>
    ),
    ItemIndicator: ({ children }: { children: ReactNode }) => <>{children}</>,
    ItemText: ({ children }: { children: ReactNode }) => <>{children}</>,
  },
}));

describe("SelectContent", () => {
  afterEach(() => {
    selectMocks.positionerProps.length = 0;
    document.body.innerHTML = "";
  });

  test("opens as a dropdown instead of overlapping the trigger", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <Select>
          <SelectContent>Role options</SelectContent>
        </Select>
      );
    });

    expect(selectMocks.positionerProps).toContainEqual(
      expect.objectContaining({
        align: "start",
        alignItemWithTrigger: false,
      })
    );

    await act(async () => {
      root.unmount();
    });
  });

  test("allows positioner props to customize dropdown placement", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <Select>
          <SelectContent
            positionerProps={{
              align: "end",
              className: "custom-positioner",
              sideOffset: 8,
            }}
          >
            Role options
          </SelectContent>
        </Select>
      );
    });

    expect(selectMocks.positionerProps).toContainEqual(
      expect.objectContaining({
        align: "end",
        alignItemWithTrigger: false,
        className: expect.stringContaining("custom-positioner"),
        sideOffset: 8,
      })
    );

    await act(async () => {
      root.unmount();
    });
  });
});

describe("SelectTrigger", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("uses pointer cursor", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <Select>
          <SelectTrigger>Role</SelectTrigger>
        </Select>
      );
    });

    expect(container.querySelector("button")?.className).toContain(
      "cursor-pointer"
    );

    await act(async () => {
      root.unmount();
    });
  });
});

describe("SelectItem", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("uses shared menu row classes for options", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <Select>
          <SelectContent>
            <SelectItem value="alpha">Alpha</SelectItem>
          </SelectContent>
        </Select>
      );
    });

    const option = container.querySelector("[role='option']");
    expect(option?.className).toContain(
      stylex.props(menuRowStyle("sm")).className ?? ""
    );
    expect(option?.className).toContain(menuRowStateClassName);

    await act(async () => {
      root.unmount();
    });
  });
});
