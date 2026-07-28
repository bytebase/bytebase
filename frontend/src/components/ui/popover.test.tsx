import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { Popover, PopoverContent, PopoverTrigger } from "./popover";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("Popover", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders trigger in container", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <Popover>
          <PopoverTrigger>
            <button type="button">Open</button>
          </PopoverTrigger>
          <PopoverContent>Popover body</PopoverContent>
        </Popover>
      );
    });

    expect(container.querySelector("button")?.textContent).toBe("Open");

    await act(async () => {
      root.unmount();
    });
  });

  test("clicking trigger opens popover content in overlay layer", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <Popover>
          <PopoverTrigger>
            <button type="button">Open</button>
          </PopoverTrigger>
          {/* Popover content renders via portal into bb-react-layer-overlay.
              We query document.body since the popup lives outside `container`. */}
          <PopoverContent>Popover body</PopoverContent>
        </Popover>
      );
    });

    const trigger = container.querySelector("button");
    expect(trigger).toBeInstanceOf(HTMLButtonElement);

    await act(async () => {
      trigger?.click();
    });

    expect(document.body.textContent).toContain("Popover body");

    await act(async () => {
      root.unmount();
    });
  });

  test("controlled popover only shows content when open=true", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    function ControlledPopover({ open }: { open: boolean }) {
      return (
        <Popover open={open}>
          <PopoverTrigger>
            <button type="button">Open</button>
          </PopoverTrigger>
          <PopoverContent>Controlled content</PopoverContent>
        </Popover>
      );
    }

    await act(async () => {
      root.render(<ControlledPopover open={false} />);
    });

    expect(document.body.textContent).not.toContain("Controlled content");

    await act(async () => {
      root.render(<ControlledPopover open={true} />);
    });

    expect(document.body.textContent).toContain("Controlled content");

    await act(async () => {
      root.unmount();
    });
  });
});
