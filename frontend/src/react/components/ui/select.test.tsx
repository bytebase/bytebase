import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Select, SelectContent } from "./select";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const selectMocks = vi.hoisted(() => ({
  positionerProps: [] as Array<Record<string, unknown>>,
}));

vi.mock("@base-ui/react/select", () => ({
  Select: {
    Root: ({ children }: { children: ReactNode }) => <div>{children}</div>,
    Trigger: ({ children }: { children: ReactNode }) => (
      <button type="button">{children}</button>
    ),
    Icon: ({ children }: { children: ReactNode }) => <>{children}</>,
    Value: ({ children }: { children: ReactNode }) => <>{children}</>,
    Portal: ({ children }: { children: ReactNode }) => <>{children}</>,
    Positioner: ({ children, ...props }: { children: ReactNode }) => {
      selectMocks.positionerProps.push(props);
      return <div>{children}</div>;
    },
    Popup: ({ children }: { children: ReactNode }) => <div>{children}</div>,
    Item: ({ children }: { children: ReactNode }) => <div>{children}</div>,
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
