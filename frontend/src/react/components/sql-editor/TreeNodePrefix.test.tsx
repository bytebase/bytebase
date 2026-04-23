import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import type { WorksheetFolderNode } from "@/views/sql-editor/Sheet";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// Stub ResizeObserver — not provided by jsdom
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

const makeNode = (
  overrides?: Partial<WorksheetFolderNode>
): WorksheetFolderNode => ({
  key: "/my/folder",
  label: "folder",
  editable: true,
  children: [],
  empty: false,
  ...overrides,
});

let TreeNodePrefix: typeof import("./TreeNodePrefix").TreeNodePrefix;

const renderIntoContainer = (element: React.ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  ({ TreeNodePrefix } = await import("./TreeNodePrefix"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("TreeNodePrefix", () => {
  test("renders FileCode icon when node has a worksheet property", () => {
    const node = makeNode({
      worksheet: {
        name: "worksheets/ws1",
        title: "My Query",
        folders: [],
        type: "worksheet",
      },
    });
    const { container, render, unmount } = renderIntoContainer(
      <TreeNodePrefix node={node} isOpen={false} rootPath="/my" view="my" />
    );
    render();

    // lucide-react renders SVGs with className "lucide-<icon-name>"
    const svg = container.querySelector("svg");
    expect(svg).not.toBeNull();
    expect(svg?.classList.contains("lucide-file-code")).toBe(true);

    unmount();
  });

  test("renders FolderOpen icon when node is expanded (id in expandedIds)", () => {
    const node = makeNode({ key: "/my/folder" });
    const _expandedIds = new Set(["/my/folder"]);

    const { container, render, unmount } = renderIntoContainer(
      <TreeNodePrefix node={node} isOpen={true} rootPath="/my" view="my" />
    );
    render();

    const svg = container.querySelector("svg");
    expect(svg).not.toBeNull();
    expect(svg?.classList.contains("lucide-folder-open")).toBe(true);

    unmount();
  });

  test("renders view-specific root icon when node.key === rootPath", () => {
    const rootPath = "/my";
    const node = makeNode({ key: rootPath });

    // my view → FolderCode (default)
    const {
      container: c1,
      render: r1,
      unmount: u1,
    } = renderIntoContainer(
      <TreeNodePrefix
        node={node}
        isOpen={false}
        rootPath={rootPath}
        view="my"
      />
    );
    r1();
    expect(
      c1.querySelector("svg")?.classList.contains("lucide-folder-code")
    ).toBe(true);
    u1();

    // draft view → FolderPen
    const {
      container: c2,
      render: r2,
      unmount: u2,
    } = renderIntoContainer(
      <TreeNodePrefix
        node={node}
        isOpen={false}
        rootPath={rootPath}
        view="draft"
      />
    );
    r2();
    expect(
      c2.querySelector("svg")?.classList.contains("lucide-folder-pen")
    ).toBe(true);
    u2();

    // shared view → FolderSync
    const {
      container: c3,
      render: r3,
      unmount: u3,
    } = renderIntoContainer(
      <TreeNodePrefix
        node={node}
        isOpen={false}
        rootPath={rootPath}
        view="shared"
      />
    );
    r3();
    expect(
      c3.querySelector("svg")?.classList.contains("lucide-folder-sync")
    ).toBe(true);
    u3();
  });
});
