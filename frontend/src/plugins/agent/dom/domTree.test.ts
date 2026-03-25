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
    expect(tree).toContain("[e1]<div>Prod Primary us-east-1</div>");
    expect(tree).toContain("  Prod Primary");
    expect(tree).toContain("  us-east-1");
    expect(getElementByRef("e1")?.tag).toBe("div");
  });

  test("does not promote inherited pointer-cursor wrappers beneath clickable containers", () => {
    document.body.innerHTML = `
      <main>
        <div style="cursor: pointer">
          <span>Prod Primary</span>
          <div>
            <span>Healthy</span>
          </div>
        </div>
      </main>
    `;

    const { tree, count } = extractDomTree();
    const suggestions = extractDomRefSuggestions();

    expect(count).toBe(1);
    expect(tree).toContain("[e1]<div>Prod Primary Healthy</div>");
    expect(tree).not.toContain("[e2]");
    expect(suggestions).toEqual([
      {
        ref: "e1",
        tag: "div",
        role: undefined,
        label: "Prod Primary Healthy",
        value: undefined,
      },
    ]);
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
    expect(tree).toContain("[e1]<div>Prod Primary us-east-1</div>");
    expect(tree.match(/Prod Primary us-east-1/g)).toHaveLength(1);
  });

  test("suppresses blank button refs while keeping meaningful native controls", () => {
    document.body.innerHTML = `
      <main>
        <button><span aria-hidden="true"></span></button>
        <button title="Refresh"></button>
        <input type="checkbox" />
        <input type="text" />
        <select>
          <option selected>Prod</option>
        </select>
      </main>
    `;

    const { tree, count } = extractDomTree();
    const suggestions = extractDomRefSuggestions();

    expect(count).toBe(4);
    expect(tree).toContain("[e1]<button>Refresh</button>");
    expect(tree).toContain('[e2]<input value="unchecked">checkbox</input>');
    expect(tree).toContain("[e3]<input>textbox</input>");
    expect(tree).toContain('[e4]<select value="Prod">Prod</select>');
    expect(tree).not.toContain("<button></button>");
    expect(suggestions).toEqual([
      {
        ref: "e1",
        tag: "button",
        role: undefined,
        label: "Refresh",
        value: undefined,
      },
      {
        ref: "e2",
        tag: "input",
        role: undefined,
        label: "checkbox",
        value: "unchecked",
      },
      {
        ref: "e3",
        tag: "input",
        role: undefined,
        label: "textbox",
        value: undefined,
      },
      {
        ref: "e4",
        tag: "select",
        role: undefined,
        label: "Prod",
        value: "Prod",
      },
    ]);
  });

  test("serializes tables with semantic header and row separators", () => {
    document.body.innerHTML = `
      <main>
        <table>
          <thead>
            <tr>
              <th>Instance</th>
              <th>Environment</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><div><span>Prod Primary</span></div></td>
              <td><span>us-east-1</span></td>
              <td><span>Healthy</span></td>
            </tr>
            <tr>
              <td><a href="/instances/staging">Staging</a></td>
              <td>us-west-2</td>
              <td><button aria-label="Open action menu">Actions</button></td>
            </tr>
          </tbody>
        </table>
      </main>
    `;

    const { tree, count } = extractDomTree();

    expect(count).toBe(2);
    expect(tree).toContain("<table>");
    expect(tree).toContain("<thead>Instance | Environment | Status</thead>");
    expect(tree).toContain("<tr>Prod Primary | us-east-1 | Healthy</tr>");
    expect(tree).toContain("<tr>Staging | us-west-2 | Actions</tr>");
    expect(tree).toContain("[e1]<a>Staging</a>");
    expect(tree).toContain("[e2]<button>Open action menu</button>");
    expect(tree).not.toContain("<div>Prod Primary");
    expect(tree).not.toContain("<span>Healthy");
  });

  test("keeps clickable table rows while suppressing wrapper descendants", () => {
    document.body.innerHTML = `
      <main>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Enabled</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            <tr style="cursor: pointer">
              <td><div><span>Prod Primary</span></div></td>
              <td>
                <div class="n-switch n-switch--active">
                  <div class="n-switch__rail"></div>
                </div>
              </td>
              <td>
                <div>
                  <button aria-label="More actions"></button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </main>
    `;

    const { tree, count } = extractDomTree();
    const suggestions = extractDomRefSuggestions();

    expect(count).toBe(3);
    expect(tree).toContain("<thead>Name | Enabled | Action</thead>");
    expect(tree).toContain("[e1]<tr>Prod Primary | switch | More actions</tr>");
    expect(tree).toContain('[e2]<div value="checked">switch</div>');
    expect(tree).toContain("[e3]<button>More actions</button>");
    expect(tree).not.toContain("n-switch__rail");
    expect(tree).not.toContain("Prod PrimaryProd Primary");
    expect(suggestions).toEqual([
      {
        ref: "e1",
        tag: "tr",
        role: undefined,
        label: "Prod Primary | switch | More actions",
        value: undefined,
      },
      {
        ref: "e2",
        tag: "div",
        role: undefined,
        label: "switch",
        value: "checked",
      },
      {
        ref: "e3",
        tag: "button",
        role: undefined,
        label: "More actions",
        value: undefined,
      },
    ]);
  });

  test("prefers clickable list items over nested wrapper clones with the same label", () => {
    document.body.innerHTML = `
      <main>
        <ul>
          <li style="cursor: pointer">
            <div role="button">
              <span>Prod Primary</span>
            </div>
          </li>
        </ul>
      </main>
    `;

    const { tree, count } = extractDomTree();
    const suggestions = extractDomRefSuggestions();

    expect(count).toBe(1);
    expect(tree).toContain("[e1]<li>Prod Primary</li>");
    expect(tree).not.toContain("[e2]");
    expect(tree.match(/Prod Primary/g)).toHaveLength(1);
    expect(suggestions).toEqual([
      {
        ref: "e1",
        tag: "li",
        role: undefined,
        label: "Prod Primary",
        value: undefined,
      },
    ]);
  });

  test("drops nested wrapper refs when a clickable row already captures the same content", () => {
    document.body.innerHTML = `
      <main>
        <table>
          <tbody>
            <tr style="cursor: pointer">
              <td>
                <div role="button">
                  <span>Prod Primary</span>
                </div>
              </td>
              <td><span>Healthy</span></td>
            </tr>
          </tbody>
        </table>
      </main>
    `;

    const { tree, count } = extractDomTree();
    const suggestions = extractDomRefSuggestions();

    expect(count).toBe(1);
    expect(tree).toContain("[e1]<tr>Prod Primary | Healthy</tr>");
    expect(tree).not.toContain("[e2]<div>Prod Primary</div>");
    expect(tree.match(/Prod Primary/g)).toHaveLength(1);
    expect(suggestions).toEqual([
      {
        ref: "e1",
        tag: "tr",
        role: undefined,
        label: "Prod Primary | Healthy",
        value: undefined,
      },
    ]);
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
    const textBlocks = Array.from(
      { length: 140 },
      (_, index) => `
      <section>
        <p>Context line ${index}</p>
        <div>
          <span>${"detail".repeat(12)} ${index}</span>
          <span>${"detail".repeat(12)} duplicate ${index}</span>
        </div>
      </section>
    `
    ).join("");
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

  test("prioritizes main content over shell regions when the budget is hit", () => {
    const headerButtons = Array.from(
      { length: 60 },
      (_, index) => `<button>Header action ${index}</button>`
    ).join("");
    const sidebarLinks = Array.from(
      { length: 60 },
      (_, index) => `<a href="/sidebar/${index}">Sidebar item ${index}</a>`
    ).join("");
    const tableRows = Array.from(
      { length: 25 },
      (_, index) => `
        <tr>
          <td>Instance ${index}</td>
          <td>Prod</td>
          <td>Healthy</td>
        </tr>
      `
    ).join("");

    document.body.innerHTML = `
      <div data-label="bb-main-body-wrapper">
        <nav data-label="bb-dashboard-header">${headerButtons}</nav>
        <div>
          <aside data-label="bb-dashboard-static-sidebar">${sidebarLinks}</aside>
          <main id="bb-layout-main">
            <button>Create instance</button>
            <table>
              <thead>
                <tr>
                  <th>Instance</th>
                  <th>Environment</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                ${tableRows}
              </tbody>
            </table>
          </main>
        </div>
      </div>
    `;

    const { tree } = extractDomTree();

    expect(tree).toMatch(/\[e\d+\]<button>Create instance<\/button>/);
    expect(tree).toContain("<thead>Instance | Environment | Status</thead>");
    expect(tree).toContain("<tr>Instance 24 | Prod | Healthy</tr>");
    expect(tree).toContain("Sidebar item 0");
    expect(tree).not.toContain("Header action 59");
    expect(tree).not.toContain("Sidebar item 59");
  });

  test("demotes overlay drawers behind main content when the budget is hit", () => {
    const drawerButtons = Array.from(
      { length: 80 },
      (_, index) => `<button>Drawer action ${index}</button>`
    ).join("");
    const rows = Array.from(
      { length: 20 },
      (_, index) => `
        <tr>
          <td>Database ${index}</td>
          <td>Production</td>
          <td>Ready</td>
        </tr>
      `
    ).join("");

    document.body.innerHTML = `
      <div data-label="bb-main-body-wrapper">
        <main id="bb-layout-main">
          <button aria-label="Create database">Create</button>
          <table>
            <thead>
              <tr>
                <th>Database</th>
                <th>Environment</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              ${rows}
            </tbody>
          </table>
        </main>
        <div class="n-drawer" role="dialog" aria-modal="true">
          ${drawerButtons}
        </div>
      </div>
    `;

    const { tree } = extractDomTree();

    expect(tree).toMatch(/\[e\d+\]<button>Create database<\/button>/);
    expect(tree).toContain("<thead>Database | Environment | Status</thead>");
    expect(tree).toContain("<tr>Database 19 | Production | Ready</tr>");
    expect(tree).toContain("Drawer action 0");
    expect(tree).not.toContain("Drawer action 79");
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
