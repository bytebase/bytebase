import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { TreeDataNode } from "./tree";
import { Tree } from "./tree";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// Stub ResizeObserver — not provided by jsdom
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

interface TestPayload {
  label: string;
}

const makeNode = (
  id: string,
  label: string,
  children?: TreeDataNode<TestPayload>[]
): TreeDataNode<TestPayload> => ({
  id,
  data: { label },
  children,
});

describe("Tree", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => {
      root.unmount();
    });
    document.body.removeChild(container);
  });

  test("a node with a falsy id renders a contained fallback instead of crashing", async () => {
    // react-arborist throws on falsy ids. The Tree must contain that
    // failure to the pane (fallback + console.error) instead of letting
    // the exception unmount the whole app.
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    try {
      const nodes = [makeNode("", "Broken")];
      await act(async () => {
        root.render(
          <Tree
            data={nodes}
            renderNode={({ node, style }) => (
              <div style={style}>{node.data.data.label}</div>
            )}
            height={200}
          />
        );
      });
      expect(container.textContent).toContain("common.render-failed");
    } finally {
      consoleError.mockRestore();
    }
  });

  test("recovers from the fallback when the data prop changes", async () => {
    // After a contained crash, swapping in new data (e.g. the user picks
    // another database) must re-attempt rendering, not stay broken.
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    try {
      const renderNode = ({
        node,
        style,
      }: {
        node: { data: TreeDataNode<TestPayload> };
        style: React.CSSProperties;
      }) => <div style={style}>{node.data.data.label}</div>;
      await act(async () => {
        root.render(
          <Tree
            data={[makeNode("", "Broken")]}
            renderNode={renderNode}
            height={200}
          />
        );
      });
      expect(container.textContent).toContain("common.render-failed");

      await act(async () => {
        root.render(
          <Tree
            data={[makeNode("ok", "Recovered")]}
            renderNode={renderNode}
            height={200}
          />
        );
      });
      expect(container.textContent).toContain("Recovered");
    } finally {
      consoleError.mockRestore();
    }
  });

  test("renders nodes from data via renderNode", async () => {
    const nodes = [makeNode("a", "Alpha"), makeNode("b", "Beta")];

    await act(async () => {
      root.render(
        <Tree
          data={nodes}
          renderNode={({ node, style }) => (
            <div style={style} data-testid={`node-${node.id}`}>
              {node.data.data.label}
            </div>
          )}
          height={200}
        />
      );
    });

    expect(container.textContent).toContain("Alpha");
    expect(container.textContent).toContain("Beta");
  });

  test("clicking a row fires onSelect with that row's id", async () => {
    const nodes = [makeNode("x", "X Node"), makeNode("y", "Y Node")];
    const handleSelect = vi.fn();

    await act(async () => {
      root.render(
        <Tree
          data={nodes}
          renderNode={({ node, style }) => (
            <div style={style} data-testid={`node-${node.id}`}>
              {node.data.data.label}
            </div>
          )}
          onSelect={handleSelect}
          height={200}
        />
      );
    });

    // Click the rendered node by finding it in the container
    const xNode = container.querySelector('[data-testid="node-x"]');
    expect(xNode).not.toBeNull();

    await act(async () => {
      xNode!.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(handleSelect).toHaveBeenCalledWith(expect.arrayContaining(["x"]));
  });

  test("selectedIds prop reflects selection via node.isSelected in renderNode", async () => {
    const nodes = [makeNode("sel", "Selected"), makeNode("other", "Other")];
    const isSelectedMap: Record<string, boolean> = {};

    await act(async () => {
      root.render(
        <Tree
          data={nodes}
          selectedIds={["sel"]}
          renderNode={({ node, style }) => {
            isSelectedMap[node.id] = node.isSelected;
            return (
              <div
                style={style}
                data-testid={`node-${node.id}`}
                data-selected={node.isSelected ? "true" : "false"}
              >
                {node.data.data.label}
              </div>
            );
          }}
          height={200}
        />
      );
    });

    // Wait for render cycle to complete and check selection via DOM
    const selNode = container.querySelector('[data-testid="node-sel"]');
    expect(selNode).not.toBeNull();
    // The node should reflect isSelected via the data-selected attribute
    // OR via react-arborist's internal selection state after render
    expect(isSelectedMap["sel"]).toBe(true);
    expect(isSelectedMap["other"]).toBe(false);
  });

  test("expandedIds controls open/close of children", async () => {
    const nodes = [
      makeNode("root", "Root", [
        makeNode("child1", "Child One"),
        makeNode("child2", "Child Two"),
      ]),
    ];

    // Render first without expanded IDs — children should not be visible
    await act(async () => {
      root.render(
        <Tree
          data={nodes}
          expandedIds={[]}
          renderNode={({ node, style }) => (
            <div style={style} data-testid={`node-${node.id}`}>
              {node.data.data.label}
            </div>
          )}
          height={200}
        />
      );
    });

    expect(container.textContent).not.toContain("Child One");

    // Now expand the root node
    await act(async () => {
      root.render(
        <Tree
          data={nodes}
          expandedIds={["root"]}
          renderNode={({ node, style }) => (
            <div style={style} data-testid={`node-${node.id}`}>
              {node.data.data.label}
            </div>
          )}
          height={200}
        />
      );
    });

    expect(container.textContent).toContain("Child One");
    expect(container.textContent).toContain("Child Two");
  });

  test("searchTerm with searchMatch filters visible nodes", async () => {
    const nodes = [
      makeNode("foo", "Foobar"),
      makeNode("baz", "Bazqux"),
      makeNode("fooAlso", "Foo Also"),
    ];

    await act(async () => {
      root.render(
        <Tree
          data={nodes}
          searchTerm="foo"
          searchMatch={(node, term) =>
            node.data.label.toLowerCase().includes(term.toLowerCase())
          }
          renderNode={({ node, style }) => (
            <div style={style} data-testid={`node-${node.id}`}>
              {node.data.data.label}
            </div>
          )}
          height={200}
        />
      );
    });

    // Nodes matching "foo" should be present; "Bazqux" should not appear
    expect(container.textContent).toContain("Foobar");
    expect(container.textContent).toContain("Foo Also");
    expect(container.textContent).not.toContain("Bazqux");
  });

  test("searchTerm can show direct children of matching nodes without recursively expanding them", async () => {
    const nodes = [
      makeNode("schema", "billing", [
        makeNode("table", "invoice", [makeNode("column", "amount")]),
        makeNode("view", "payment_view"),
      ]),
      makeNode("audit", "audit", [makeNode("log", "log_entry")]),
    ];

    await act(async () => {
      root.render(
        <Tree
          data={nodes}
          searchTerm="billing"
          includeChildrenOnSearchMatch
          searchMatch={(node, term) =>
            node.data.label.toLowerCase().includes(term.toLowerCase())
          }
          renderNode={({ node, style }) => (
            <div style={style} data-testid={`node-${node.id}`}>
              {node.data.data.label}
            </div>
          )}
          height={200}
        />
      );
    });

    expect(container.textContent).toContain("billing");
    expect(container.textContent).toContain("invoice");
    expect(container.textContent).toContain("payment_view");
    expect(container.textContent).not.toContain("amount");
    expect(container.textContent).not.toContain("audit");
    expect(container.textContent).not.toContain("log_entry");
  });
});
