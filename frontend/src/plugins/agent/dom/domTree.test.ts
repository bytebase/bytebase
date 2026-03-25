import { afterEach, describe, expect, test } from "vitest";
import {
  extractDomRefSuggestions,
  extractDomTree,
  getElementByRef,
} from "./domTree";

afterEach(() => {
  document.body.innerHTML = "";
});

describe("extractDomTree", () => {
  test("emits eN refs for interactive elements and Monaco editors", () => {
    document.body.innerHTML = `
      <main>
        <button aria-label="Run query">Run</button>
        <input placeholder="Project name" value="Test Project" />
        <div class="monaco-editor">
          <div class="view-lines">
            <div class="view-line">SELECT *</div>
            <div class="view-line">FROM projects;</div>
          </div>
        </div>
      </main>
    `;

    const { tree, count } = extractDomTree();

    expect(count).toBe(3);
    expect(tree).toContain("[e1]<button>Run query</button>");
    expect(tree).toContain(
      '[e2]<input value="Test Project">Project name</input>'
    );
    expect(tree).toContain(
      "[e3]<editor>SQL: SELECT *\nFROM projects;</editor>"
    );
    expect(getElementByRef("e1")?.tag).toBe("button");
    expect(getElementByRef("e3")?.tag).toBe("editor");
    expect(getElementByRef("e9")).toBeUndefined();
  });

  test("preserves visible non-interactive text nodes with normalized whitespace", () => {
    document.body.innerHTML = `
      <main>
        <section>
          <h1>   Instances   </h1>
          <div>
            <span>Primary</span>
            <span>US East</span>
          </div>
          <button aria-label="Create instance">Create</button>
          <p style="display: none">Hidden text</p>
          <div aria-hidden="true" style="display: none">
            <span>Also hidden</span>
          </div>
        </section>
      </main>
    `;

    const { tree, count } = extractDomTree();

    expect(count).toBe(1);
    expect(tree).toContain("Instances");
    expect(tree).toContain("Primary");
    expect(tree).toContain("US East");
    expect(tree).toContain("[e1]<button>Create instance</button>");
    expect(tree).not.toContain("Hidden text");
    expect(tree).not.toContain("Also hidden");
  });

  test("treats clickable containers with pointer cursor as interactive and preserves descendant text", () => {
    document.body.innerHTML = `
      <main>
        <div style="cursor: pointer">
          <span>Prod Primary</span>
          <span>us-east-1</span>
        </div>
      </main>
    `;

    const { tree, count } = extractDomTree();

    expect(count).toBe(1);
    expect(tree).toContain(
      "[e1]<div>Prod Primary us-east-1</div>"
    );
    expect(tree).toContain("  Prod Primary");
    expect(tree).toContain("  us-east-1");
    expect(getElementByRef("e1")?.tag).toBe("div");
  });

  test("skips disabled and hidden pointer-cursor containers", () => {
    document.body.innerHTML = `
      <main>
        <div style="cursor: pointer" disabled>Disabled row</div>
        <div style="cursor: pointer" aria-disabled="true">Aria disabled row</div>
        <div style="cursor: pointer" inert>Inert row</div>
        <div style="cursor: pointer" aria-hidden="true">Aria hidden row</div>
        <div style="cursor: pointer">Active row</div>
      </main>
    `;

    const { tree, count } = extractDomTree();
    const suggestions = extractDomRefSuggestions();

    expect(count).toBe(1);
    expect(tree).toContain("[e1]<div>Active row</div>");
    expect(tree).toContain("Disabled row");
    expect(tree).toContain("Aria disabled row");
    expect(tree).not.toContain("Inert row");
    expect(tree).not.toContain("Aria hidden row");
    expect(suggestions).toEqual([
      {
        ref: "e1",
        tag: "div",
        role: undefined,
        label: "Active row",
        value: undefined,
      },
    ]);
  });

  test("treats contenteditable regions as interactive without duplicating descendants", () => {
    document.body.innerHTML = `
      <main>
        <div contenteditable="true">Editable SQL</div>
      </main>
    `;

    const { tree, count } = extractDomTree();

    expect(count).toBe(1);
    expect(tree).toContain("[e1]<div>Editable SQL</div>");
    expect(tree.match(/Editable SQL/g)).toHaveLength(1);
  });

  test("dedupes descendant text when it matches an interactive label", () => {
    document.body.innerHTML = `
      <main>
        <div style="cursor: pointer">
          <span>Prod Primary us-east-1</span>
        </div>
      </main>
    `;

    const { tree, count } = extractDomTree();

    expect(count).toBe(1);
    expect(tree).toContain(
      "[e1]<div>Prod Primary us-east-1</div>"
    );
    expect(tree.match(/Prod Primary us-east-1/g)).toHaveLength(1);
  });

  test("truncates long labels and values deterministically", () => {
    const longText = "x".repeat(180);
    document.body.innerHTML = `
      <main>
        <button>${longText}</button>
        <input value="${longText}" />
      </main>
    `;

    const { tree } = extractDomTree();

    expect(tree).toContain(`${"x".repeat(120)}...`);
    expect(tree).not.toContain("x".repeat(150));
  });

  test("drops plain text before interactive refs when the tree budget is hit", () => {
    const textBlocks = Array.from({ length: 140 }, (_, index) => `
      <section>
        <p>Context line ${index}</p>
        <div>
          <span>${"detail".repeat(12)} ${index}</span>
          <span>${"detail".repeat(12)} duplicate ${index}</span>
        </div>
      </section>
    `).join("");
    document.body.innerHTML = `
      <main>
        ${textBlocks}
        <button aria-label="Launch workflow">Launch</button>
      </main>
    `;

    const { tree, count } = extractDomTree();

    expect(count).toBe(1);
    expect(tree).toContain("[e1]<button>Launch workflow</button>");
    expect(tree).not.toContain("Context line 139");
    expect(tree.split("\n").length).toBeLessThanOrEqual(120);
  });

  test("returns structured DOM ref suggestions for visible interactive elements", () => {
    document.body.innerHTML = `
      <main>
        <button aria-label="Run query">Run</button>
        <div role="button" aria-labelledby="publish-label">Publish now</div>
        <span id="publish-label">Publish</span>
        <input placeholder="Project name" value="Test Project" />
        <div class="monaco-editor">
          <div class="view-lines">
            <div class="view-line">SELECT *</div>
            <div class="view-line">FROM projects;</div>
          </div>
        </div>
        <button style="display: none">Hidden</button>
      </main>
    `;

    expect(extractDomRefSuggestions()).toEqual([
      {
        ref: "e1",
        tag: "button",
        role: undefined,
        label: "Run query",
        value: undefined,
      },
      {
        ref: "e2",
        tag: "div",
        role: "button",
        label: "Publish",
        value: undefined,
      },
      {
        ref: "e3",
        tag: "input",
        role: undefined,
        label: "Project name",
        value: "Test Project",
      },
      {
        ref: "e4",
        tag: "editor",
        role: undefined,
        label: "SQL: SELECT *\nFROM projects;",
        value: "SELECT *\nFROM projects;",
      },
    ]);

    expect(getElementByRef("e2")?.label).toBe("Publish");
  });

  test("resets refs on each extraction", () => {
    document.body.innerHTML = `<button>First</button>`;
    expect(extractDomTree().tree).toContain("[e1]<button>First</button>");

    document.body.innerHTML = `<button>Second</button>`;
    const { tree, count } = extractDomTree();

    expect(count).toBe(1);
    expect(tree).toContain("[e1]<button>Second</button>");
    expect(tree).not.toContain("[e2]");
  });
});
