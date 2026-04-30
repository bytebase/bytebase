import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ColumnMetadataSchema,
  SchemaMetadataSchema,
  TableMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { Layout } from "./index";
import { useAutoLayout } from "./useAutoLayout";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const autoLayoutMock = vi.fn<(...args: unknown[]) => Promise<Layout>>();

vi.mock("./index", async (importOriginal) => {
  const orig = await importOriginal<typeof import("./index")>();
  return {
    ...orig,
    autoLayout: (...args: unknown[]) => autoLayoutMock(...args),
  };
});

const renderHook = <T,>(hookFn: () => T) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  let value!: T;
  function Host() {
    value = hookFn();
    return null;
  }
  act(() => {
    root.render(<Host />);
  });
  return {
    get current() {
      return value;
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const makeTable = (name: string) =>
  create(TableMetadataSchema, {
    name,
    columns: [create(ColumnMetadataSchema, { name: "id" })],
  });

const makeSchema = (name: string, tables: ReturnType<typeof makeTable>[]) =>
  create(SchemaMetadataSchema, { name, tables });

beforeEach(() => {
  autoLayoutMock.mockReset();
});
afterEach(() => {
  vi.clearAllMocks();
});

describe("useAutoLayout", () => {
  it("threads sizes through ELK and returns the rect map", async () => {
    const t1 = makeTable("users");
    const t2 = makeTable("orders");
    const schema = makeSchema("public", [t1, t2]);

    const idOfTable = (t: typeof t1) => `id-${t.name}`;
    const sizeOfTable = (id: string) =>
      id === "id-users"
        ? { width: 200, height: 80 }
        : { width: 240, height: 100 };

    autoLayoutMock.mockResolvedValueOnce({
      rects: new Map([
        ["id-users", { x: 0, y: 0, width: 200, height: 80 }],
        ["id-orders", { x: 300, y: 0, width: 240, height: 100 }],
      ]),
      paths: new Map(),
    });

    const hook = renderHook(() =>
      useAutoLayout({
        selectedSchemas: [schema],
        edges: [
          {
            fromSchema: "public",
            fromTable: t2,
            fromColumn: "user_id",
            toSchema: "public",
            toTable: t1,
            toColumn: "id",
          },
        ],
        idOfTable,
        sizeOfTable,
      })
    );

    let rects: Map<string, { x: number; y: number }> | null = null;
    await act(async () => {
      const out = await hook.current();
      rects = out as Map<string, { x: number; y: number }>;
    });

    expect(rects).not.toBeNull();
    expect(rects!.get("id-users")).toEqual({
      x: 0,
      y: 0,
      width: 200,
      height: 80,
    });

    const callArgs = autoLayoutMock.mock.calls[0];
    const nodeList = callArgs[0] as Array<{
      id: string;
      size: { width: number; height: number };
      group: string;
    }>;
    const edgeList = callArgs[1] as Array<{
      id: string;
      from: string;
      to: string;
    }>;
    expect(nodeList).toHaveLength(2);
    expect(nodeList.map((n) => n.id).sort()).toEqual(["id-orders", "id-users"]);
    expect(nodeList[0].group).toBe("schema-public");
    expect(edgeList).toHaveLength(1);
    expect(edgeList[0].from).toBe("id-orders");
    expect(edgeList[0].to).toBe("id-users");

    hook.unmount();
  });

  it("skips tables whose DOM size is unknown", async () => {
    const t1 = makeTable("users");
    const t2 = makeTable("orders");
    const schema = makeSchema("public", [t1, t2]);

    autoLayoutMock.mockResolvedValueOnce({
      rects: new Map(),
      paths: new Map(),
    });

    const hook = renderHook(() =>
      useAutoLayout({
        selectedSchemas: [schema],
        edges: [],
        idOfTable: (t) => `id-${t.name}`,
        sizeOfTable: (id) =>
          id === "id-users" ? { width: 200, height: 80 } : null,
      })
    );

    await act(async () => {
      await hook.current();
    });

    const nodeList = autoLayoutMock.mock.calls[0][0] as Array<{
      id: string;
    }>;
    expect(nodeList).toHaveLength(1);
    expect(nodeList[0].id).toBe("id-users");

    hook.unmount();
  });

  it("discards stale results when a newer call supersedes", async () => {
    const t1 = makeTable("users");
    const schema = makeSchema("public", [t1]);

    let resolveA: (layout: Layout) => void = () => {};
    autoLayoutMock.mockImplementationOnce(
      () =>
        new Promise<Layout>((resolve) => {
          resolveA = resolve;
        })
    );
    autoLayoutMock.mockResolvedValueOnce({
      rects: new Map([["id-users", { x: 999, y: 0, width: 1, height: 1 }]]),
      paths: new Map(),
    });

    const hook = renderHook(() =>
      useAutoLayout({
        selectedSchemas: [schema],
        edges: [],
        idOfTable: (t) => `id-${t.name}`,
        sizeOfTable: () => ({ width: 100, height: 50 }),
      })
    );

    let firstResult: Map<string, unknown> | null = null;
    let secondResult: Map<string, unknown> | null = null;

    await act(async () => {
      const a = hook.current();
      const b = hook.current();
      // Resolve A *after* B has been kicked off, so A is now stale.
      resolveA({
        rects: new Map([["id-users", { x: 0, y: 0, width: 1, height: 1 }]]),
        paths: new Map(),
      });
      firstResult = (await a) as Map<string, unknown> | null;
      secondResult = (await b) as Map<string, unknown> | null;
    });

    expect(firstResult).toBeNull();
    expect(secondResult).not.toBeNull();
    expect(secondResult!.get("id-users")).toEqual({
      x: 999,
      y: 0,
      width: 1,
      height: 1,
    });

    hook.unmount();
  });
});
