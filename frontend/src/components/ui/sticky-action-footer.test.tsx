import * as stylex from "@stylexjs/stylex";
import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { StickyActionFooter } from "./sticky-action-footer";
import {
  stickyActionFooterContentStyle,
  stickyActionFooterRightStyle,
  stickyActionFooterSideStyle,
  stickyActionFooterStyle,
} from "./styles.stylex";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

describe("StickyActionFooter", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("keeps negative actions on the left and positive actions on the right", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(StickyActionFooter, {
        left: createElement("button", { type: "button" }, "Cancel"),
        right: [
          createElement("button", { key: "test", type: "button" }, "Test"),
          createElement("button", { key: "save", type: "button" }, "Save"),
        ],
      })
    );

    const footer = container.querySelector(
      '[data-slot="sticky-action-footer"]'
    );
    const left = container.querySelector(
      '[data-slot="sticky-action-footer-left"]'
    );
    const right = container.querySelector(
      '[data-slot="sticky-action-footer-right"]'
    );
    const content = container.querySelector(
      '[data-slot="sticky-action-footer-content"]'
    );

    expect(footer?.className).toContain("sticky");
    expect(footer?.className).toContain("bottom-0");
    expect(footer?.className).toContain(
      stylex.props(stickyActionFooterStyle()).className ?? ""
    );
    expect(footer?.firstElementChild).toBe(content);
    expect(content?.className).toContain("px-4");
    expect(content?.className).toContain("sm:px-6");
    expect(content?.className).toContain(
      stylex.props(stickyActionFooterContentStyle()).className ?? ""
    );
    expect(content?.firstElementChild).toBe(left);
    expect(content?.lastElementChild).toBe(right);
    expect(left?.textContent).toBe("Cancel");
    expect(left?.className).toContain(
      stylex.props(stickyActionFooterSideStyle()).className ?? ""
    );
    expect(right?.textContent).toBe("TestSave");
    expect(right?.className).toContain("gap-x-2");
    expect(right?.className).toContain(
      stylex.props(
        stickyActionFooterSideStyle(),
        stickyActionFooterRightStyle()
      ).className ?? ""
    );

    unmount();
  });
});
