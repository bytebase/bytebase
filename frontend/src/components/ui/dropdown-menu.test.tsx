import * as stylex from "@stylexjs/stylex";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "./dropdown-menu";
import {
  menuRowStateClassName,
  menuRowStyle,
  overlaySurfaceClassName,
} from "./styles.stylex";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

globalThis.ResizeObserver ??= class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

describe("DropdownMenu", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("uses shared menu row state classes for items", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>Action</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      );
    });

    await act(async () => {
      container
        .querySelector("button")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain("Action");
    });

    const item = Array.from(
      document.querySelectorAll("[role='menuitem']")
    ).find((el) => el.textContent === "Action");
    const surface = item?.parentElement;
    expect(surface?.className).toContain(overlaySurfaceClassName);
    expect(item?.className).toContain(
      stylex.props(menuRowStyle("sm")).className ?? ""
    );
    expect(item?.className).toContain(menuRowStateClassName);

    await act(async () => {
      root.unmount();
    });
  });

  test("renders section labels for grouped menus", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <DropdownMenu>
          <DropdownMenuTrigger>Open</DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuLabel>Section</DropdownMenuLabel>
            <DropdownMenuItem>Action</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      );
    });

    await act(async () => {
      container
        .querySelector("button")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain("Section");
    });

    const label = Array.from(document.querySelectorAll("div")).find(
      (el) => el.textContent === "Section"
    );
    expect(label?.className).toContain("text-xs");
    expect(label?.className).toContain("text-control-light");

    await act(async () => {
      root.unmount();
    });
  });
});
