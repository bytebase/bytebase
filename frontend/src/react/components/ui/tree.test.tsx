import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { TreeDataNode } from "./tree";
import { Tree } from "./tree";

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
});
