import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ColumnMetadataSchema,
  DatabaseMetadataSchema,
  DatabaseSchema$,
  SchemaMetadataSchema,
  TableMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { Layout } from "./ER/libs/autoLayout";
import { SchemaDiagram } from "./SchemaDiagram";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const autoLayoutMock = vi.fn<(...args: unknown[]) => Promise<Layout>>();

vi.mock("./ER/libs/autoLayout/index", async (importOriginal) => {
  const orig =
    await importOriginal<typeof import("./ER/libs/autoLayout/index")>();
  return {
    ...orig,
    autoLayout: (...args: unknown[]) => autoLayoutMock(...args),
  };
});

vi.mock("./Navigator/Navigator", () => ({ Navigator: () => null }));

const CANVAS = { width: 1000, height: 600 };

const TABLE_RECTS = new Map([
  ["t0", { x: 0, y: 0, width: 256, height: 150 }],
  ["t1", { x: 900, y: 0, width: 256, height: 150 }],
  ["t2", { x: 1800, y: 400, width: 256, height: 150 }],
]);

const makeMetadata = () =>
  create(DatabaseMetadataSchema, {
    name: "instances/i/databases/db/metadata",
    schemas: [
      create(SchemaMetadataSchema, {
        name: "public",
        tables: ["users", "orders", "items"].map((name) =>
          create(TableMetadataSchema, {
            name,
            columns: [create(ColumnMetadataSchema, { name: "id" })],
          })
        ),
      }),
    ],
  });

const desktopTransform = (container: HTMLElement) => {
  const desktop = Array.from(container.querySelectorAll("div")).find((el) =>
    el.style.transform.startsWith("matrix(")
  );
  const [zoom, , , , x, y] = (desktop?.style.transform ?? "")
    .slice("matrix(".length, -1)
    .split(",")
    .map(Number);
  return { zoom, x, y };
};

let container: HTMLDivElement;
let root: Root;

beforeEach(() => {
  autoLayoutMock.mockReset();
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockReturnValue(
    DOMRect.fromRect(CANVAS)
  );
  // A synchronous frame callback reaches fit-view before React commits
  // the laid-out rects, the timing that loses the race on a slow host.
  vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
    cb(0);
    return 0;
  });
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("SchemaDiagram", () => {
  it("fits every laid-out table into the canvas on mount", async () => {
    autoLayoutMock.mockImplementation(async (nodes) => ({
      rects: new Map(
        (nodes as { id: string }[]).map(({ id }, i) => [
          id,
          TABLE_RECTS.get(`t${i}`)!,
        ])
      ),
      paths: new Map(),
    }));

    await act(async () => {
      root.render(
        <SchemaDiagram
          database={create(DatabaseSchema$, {
            name: "instances/i/databases/db",
          })}
          databaseMetadata={makeMetadata()}
        />
      );
    });

    expect(autoLayoutMock).toHaveBeenCalled();
    const { zoom, x, y } = desktopTransform(container);
    expect(zoom).toBeLessThan(1);
    for (const rect of TABLE_RECTS.values()) {
      expect(x + rect.x * zoom).toBeGreaterThanOrEqual(0);
      expect(y + rect.y * zoom).toBeGreaterThanOrEqual(0);
      expect(x + (rect.x + rect.width) * zoom).toBeLessThanOrEqual(
        CANVAS.width
      );
      expect(y + (rect.y + rect.height) * zoom).toBeLessThanOrEqual(
        CANVAS.height
      );
    }
  });
});
