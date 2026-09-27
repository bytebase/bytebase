import { readFileSync } from "node:fs";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { MarkdownEditor } from "./MarkdownEditor";

// Vitest stubs CSS imports; load the actual stylesheet from the frontend test root.
const markdownStyles = readFileSync("src/assets/css/github-markdown-style.css", "utf8");

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
  test.each([
    { content: "2222", lastTag: "P" },
    { content: "First paragraph\n\nLast paragraph", lastTag: "P" },
    { content: "Introduction\n\n- Item", lastTag: "UL" },
    { content: "Introduction\n\n```sql\nSELECT 1;\n```", lastTag: "PRE" },
    { content: "Introduction\n\n## Heading", lastTag: "H2" },
  ])("removes the final $lastTag margin while preserving spacing between blocks", ({ content, lastTag }) => {
    const stylesheet = document.createElement("style");
    stylesheet.textContent = markdownStyles;
    document.head.append(stylesheet);
    const { container, render, unmount } = renderIntoContainer(
      <MarkdownEditor content={content} mode="preview" />
    );
    document.body.append(container);
    try {
      render();
      const body = container.querySelector(".markdown-body")!;
      const last = body.lastElementChild!;
      expect(last.tagName).toBe(lastTag);
      expect(Number.parseFloat(getComputedStyle(last).marginBottom)).toBe(0);
      if (body.firstElementChild !== last) {
        expect(Number.parseFloat(getComputedStyle(body.firstElementChild!).marginBottom)).toBe(16);
      }
    } finally {
      unmount();
      container.remove();
      stylesheet.remove();
    }
  });

  test("switches between accessible tabs without losing the draft", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MarkdownEditor content="**draft**" onChange={vi.fn()} />
    );
    render();
    const tabs = container.querySelectorAll<HTMLButtonElement>('[role="tab"]');
    expect(tabs).toHaveLength(2);
    expect(tabs[0].getAttribute("aria-selected")).toBe("true");

    act(() => tabs[1].click());
    expect(tabs[1].getAttribute("aria-selected")).toBe("true");
    expect(container.querySelector('[role="tabpanel"] strong')?.textContent).toBe(
      "draft"
    );

    act(() => tabs[0].click());
    expect(tabs[0].getAttribute("aria-selected")).toBe("true");
    expect(container.querySelector("textarea")?.value).toBe("**draft**");
    unmount();
  });

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


test("focuses the input when autofocus is requested", () => {
  const {container, render, unmount} = renderIntoContainer(<MarkdownEditor autoFocus compact content="" onChange={vi.fn()} />);
  document.body.append(container);
  try {
    render();
    expect(document.activeElement).toBe(container.querySelector("textarea"));
  } finally {
    unmount();
    container.remove();
  }
});
