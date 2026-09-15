import * as stylex from "@stylexjs/stylex";
import { act, type CSSProperties, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Select, SelectContent, SelectItem, SelectTrigger } from "./select";
import {
  menuRowStateClassName,
  menuRowStyle,
  overlaySurfaceClassName,
} from "./styles.stylex";

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
    Popup: ({
      children,
      ...props
    }: {
      children: ReactNode;
      className?: string;
    }) => <div {...props}>{children}</div>,
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
    ItemText: ({ children }: { children: ReactNode }) => <span>{children}</span>,
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

  test("uses the shared overlay surface contract", async () => {
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

    const popup = Array.from(container.querySelectorAll("div")).find(
      (element) =>
        element.textContent === "Role options" &&
        element.className.includes("min-w-")
    );
    expect(popup?.className).toContain(overlaySurfaceClassName);

    await act(async () => {
      root.unmount();
    });
  });
});

describe("Select multi-select chips", () => {
  afterEach(cleanup);

  async function loadSelect() {
    vi.resetModules();
    vi.doUnmock("@base-ui/react/select");
    return import("./select");
  }

  test("removes a labeled chip without opening the dropdown and restores the placeholder", async () => {
    const { Select, SelectTrigger, SelectValue } = await loadSelect();
    const onValueChange = vi.fn();
    const props = {
      multiple: true as const,
      items: { email: "Email", phone: "Phone" },
      onValueChange,
    };
    const view = render(
      <Select {...props} value={["email", "phone"]}>
        <SelectTrigger aria-label="Columns">
          <SelectValue placeholder="Select columns" />
        </SelectTrigger>
      </Select>
    );

    const remove = screen.getAllByRole("button");
    expect(remove).toHaveLength(2);
    expect(remove[0]?.closest("button button")).toBeNull();
    fireEvent.pointerDown(remove[0]!, { pointerType: "mouse" });
    fireEvent.mouseDown(remove[0]!);
    fireEvent.click(remove[0]!);
    expect(onValueChange.mock.calls[0]?.[0]).toEqual(["phone"]);
    expect(screen.getByRole("combobox")).toHaveAttribute("aria-expanded", "false");

    view.rerender(
      <Select {...props} value={[]}>
        <SelectTrigger aria-label="Columns">
          <SelectValue placeholder="Select columns" />
        </SelectTrigger>
      </Select>
    );
    expect(screen.getByText("Select columns")).toBeTruthy();
    expect(screen.queryAllByRole("button")).toHaveLength(0);
  });

  test("honors cancellation of chip removal", async () => {
    const { Select, SelectTrigger, SelectValue } = await loadSelect();
    render(
      <Select
        multiple
        defaultValue={["Email"]}
        onValueChange={(_, details) => details.cancel()}
      >
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
      </Select>
    );
    fireEvent.click(screen.getByRole("button"));
    expect(screen.getByText("Email")).toBeTruthy();
  });

  test("supports removing uncontrolled values and hides removal when disabled or read-only", async () => {
    const { Select, SelectTrigger, SelectValue } = await loadSelect();
    const view = render(
      <Select multiple defaultValue={["Email", "Phone"]}>
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
      </Select>
    );
    fireEvent.click(screen.getAllByRole("button")[0]!);
    expect(screen.queryByText("Email")).toBeNull();
    expect(screen.getByText("Phone")).toBeTruthy();

    for (const state of [{ disabled: true }, { readOnly: true }]) {
      view.rerender(
        <Select multiple value={["Phone"]} {...state}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
        </Select>
      );
      expect(screen.queryAllByRole("button")).toHaveLength(0);
      expect(screen.getByText("Phone")).toBeTruthy();
    }
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
    expect(option?.firstElementChild?.textContent).toBe("Alpha");
    expect(option?.lastElementChild?.querySelector("svg")).not.toBeNull();

    await act(async () => {
      root.unmount();
    });
  });
});
