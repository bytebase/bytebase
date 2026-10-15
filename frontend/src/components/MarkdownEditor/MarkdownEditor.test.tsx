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

  test("renders untrusted images and raw HTML without resource-loading elements or attributes", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MarkdownEditor
        content={[
          "![inline](https://attacker.example/inline.png?q=secret)",
          "![reference][logo]",
          "[logo]: https://attacker.example/reference.png",
          '<img src="https://attacker.example/raw.png">',
          '<picture><source srcset="https://attacker.example/s.png"></picture>',
          '<svg><image href="https://attacker.example/svg.png"></image></svg>',
          '<video poster="https://attacker.example/poster.png"></video>',
          '<table background="https://attacker.example/table.png"><tr><td>t</td></tr></table>',
          '<p style="background:url(https://attacker.example/bg.png)">styled</p>',
          '<style>@import "https://attacker.example/x.css";</style>',
          "&lt;img src=https://attacker.example/entity.png&gt;",
          "| a |\n|:-:|\n| b |",
          "```sql\nSELECT 1;\n```",
        ].join("\n\n")}
        mode="preview"
        untrusted
      />
    );
    render();

    expect(
      container.querySelector(
        "img, picture, source, svg, image, video, audio, iframe, object, embed, link, style, [src], [srcset], [poster], [background], [style]"
      )
    ).toBeNull();
    expect(container.querySelector("[class]:not(.markdown-body)")).toBeNull();
    const imageLinks = Array.from(container.querySelectorAll("a"))
      .filter((link) => ["inline", "reference"].includes(link.textContent ?? ""))
      .map((link) => [link.textContent, link.getAttribute("href"), link.getAttribute("target")]);
    expect(imageLinks).toEqual([
      ["inline", "https://attacker.example/inline.png?q=secret", "_blank"],
      ["reference", "https://attacker.example/reference.png", "_blank"],
    ]);
    expect(container.textContent).not.toContain("!inline");
    expect(container.textContent).toContain(
      '<img src="https://attacker.example/raw.png">'
    );
    expect(container.querySelector("td")?.textContent).toBe("b");
    expect(container.querySelector("pre code")?.textContent).toBe("SELECT 1;\n");
    unmount();
  });

  test("labels an untrusted image link with its alt text, entities and code spans included, or else its source", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MarkdownEditor
        content={[
          "![R&amp;D \\*draft\\*](https://attacker.example/a.png)",
          "![`build` log](https://attacker.example/b.png)",
          "![](https://attacker.example/c.png)",
          "[![](https://attacker.example/d.png)](https://example.com/ci)",
        ].join("\n\n")}
        mode="preview"
        untrusted
      />
    );
    render();

    const links = Array.from(container.querySelectorAll("a")).map((link) => [
      link.textContent,
      link.getAttribute("href"),
    ]);
    expect(links).toEqual([
      ["R&D *draft*", "https://attacker.example/a.png"],
      ["build log", "https://attacker.example/b.png"],
      ["https://attacker.example/c.png", "https://attacker.example/c.png"],
      ["https://attacker.example/d.png", "https://example.com/ci"],
    ]);
    unmount();
  });

  test("keeps only the alt text of an untrusted image inside a link", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MarkdownEditor
        content="[![build badge](https://attacker.example/badge.png)](https://example.com/ci)"
        mode="preview"
        untrusted
      />
    );
    render();

    const links = Array.from(container.querySelectorAll("a")).map((link) => [
      link.textContent,
      link.getAttribute("href"),
    ]);
    expect(links).toEqual([["build badge", "https://example.com/ci"]]);
    unmount();
  });

  test("keeps images and HTML in ordinary text", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MarkdownEditor
        content={"![chart](https://example.com/chart.png)\n\n<details>more</details>"}
        mode="preview"
      />
    );
    render();

    expect(container.querySelector("img")?.getAttribute("src")).toBe(
      "https://example.com/chart.png"
    );
    expect(container.querySelector("details")).not.toBeNull();
    unmount();
  });

  test("renders an untrusted draft's preview tab without images", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MarkdownEditor
        content="![chart](https://attacker.example/leak.png)"
        onChange={vi.fn()}
        untrusted
      />
    );
    render();
    const tabs = container.querySelectorAll<HTMLButtonElement>('[role="tab"]');
    act(() => tabs[1].click());

    expect(container.querySelector('[role="tabpanel"] a')).not.toBeNull();
    expect(container.querySelector("img")).toBeNull();
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
