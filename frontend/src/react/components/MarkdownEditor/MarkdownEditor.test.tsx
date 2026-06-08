import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { MarkdownEditor } from "./MarkdownEditor";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

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

describe("MarkdownEditor", () => {
  test("opens rendered markdown links in a new tab", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MarkdownEditor
        content="[docs](https://docs.bytebase.com) https://example.com"
        mode="preview"
      />
    );

    render();

    const links = Array.from(container.querySelectorAll("a"));
    expect(links).toHaveLength(2);
    for (const link of links) {
      expect(link.getAttribute("target")).toBe("_blank");
      expect(link.getAttribute("rel")).toBe("noopener noreferrer");
    }

    unmount();
  });
});
