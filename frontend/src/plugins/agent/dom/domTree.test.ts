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
