import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// Captured props from the mocked Combobox — inspected/driven by each test.
type ComboProps = {
  multiple?: boolean;
  value: string | string[];
  onChange: (v: string | string[]) => void;
  onSearch?: (q: string) => void;
  options: { value: string; label: string }[];
  renderValue?: (opt: { value: string; label: string }) => unknown;
};
const combo: { props?: ComboProps } = {};

const mocks = vi.hoisted(() => {
  // Simulated global database cache. The store upserts every fetched database
  // here, so getDatabaseByName resolves on-page and previously-seen names; a
  // miss returns an "unknown" placeholder ({ name: "" }) → synthesized label.
  const cache = new Map<
    string,
    { name: string; effectiveEnvironment?: string }
  >();
  return {
    cache,
    fetchDatabases: vi.fn(),
    workspaceResourceName: vi.fn(() => "workspaces/-"),
    getDatabaseByName: vi.fn((name: string) => cache.get(name) ?? { name: "" }),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k }),
}));

vi.mock("@/react/stores/app", () => {
  const state = () => ({
    fetchDatabases: mocks.fetchDatabases,
    workspaceResourceName: mocks.workspaceResourceName,
    getDatabaseByName: mocks.getDatabaseByName,
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

// Capture-only Combobox mock (repo convention). Renders nothing interactive.
vi.mock("@/react/components/ui/combobox", () => ({
  Combobox: (props: ComboProps) => {
    combo.props = props;
    return null;
  },
}));

// EngineIcon / EnvironmentLabel are only used inside option.render — never
// invoked by these tests — but stub them so the module imports cleanly.
vi.mock("@/react/components/EngineIcon", () => ({ EngineIcon: () => null }));
vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: () => null,
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => {
    const parts = name.split("/");
    return { databaseName: parts[3] ?? name, instance: parts[1] ?? "" };
  },
  getDatabaseEnvironment: () => ({ name: "environments/prod" }),
  getDefaultPagination: () => 50,
  getInstanceResource: () => ({ engine: 0, title: "inst" }),
}));

let DatabaseSelect: typeof import("./DatabaseSelect").DatabaseSelect;

const db = (
  name: string,
  effectiveEnvironment = "environments/prod"
): Database => ({ name, effectiveEnvironment }) as unknown as Database;

const render = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  act(() => root.render(element));
  return {
    container,
    rerender: (el: ReactElement) => act(() => root.render(el)),
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

// Let the mount-time fetch's .then() microtask resolve and re-render.
const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

beforeEach(async () => {
  vi.clearAllMocks();
  combo.props = undefined;
  mocks.fetchDatabases.mockResolvedValue({ databases: [], nextPageToken: "" });
  mocks.cache.clear();
  ({ DatabaseSelect } = await import("./DatabaseSelect"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("DatabaseSelect — multi mode", () => {
  test("passes multiple=true and returns string[] from onChange", async () => {
    const dbA = db("instances/i/databases/a");
    const dbB = db("instances/i/databases/b");
    mocks.cache.set(dbA.name, dbA);
    mocks.cache.set(dbB.name, dbB);
    mocks.fetchDatabases.mockResolvedValue({
      databases: [dbA, dbB],
      nextPageToken: "",
    });
    const onChange = vi.fn();
    const { unmount } = render(
      <DatabaseSelect
        multiple
        value={[]}
        onChange={onChange}
        projectName="projects/p"
      />
    );
    await flush();

    expect(combo.props?.multiple).toBe(true);
    act(() => combo.props?.onChange(["instances/i/databases/a"]));
    expect(onChange).toHaveBeenCalledWith(
      ["instances/i/databases/a"],
      expect.arrayContaining([
        expect.objectContaining({ name: "instances/i/databases/a" }),
      ])
    );
    unmount();
  });

  // GREEN GUARD: the initial load fetches the first page with an empty query.
  test("mount fetches the first page with an empty query (guard)", async () => {
    const { unmount } = render(
      <DatabaseSelect
        multiple
        value={[]}
        onChange={vi.fn()}
        projectName="projects/p"
      />
    );
    await flush();
    expect(mocks.fetchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "projects/p",
        filter: expect.objectContaining({ query: "" }),
      })
    );
    unmount();
  });

  // GREEN GUARD (passes pre- and post-fix). Not the bug repro — DatabaseSelect
  // already wires onSearch; the missing onSearch was in the DRAWER's local
  // MultiDatabaseSelect. The true red bug-repro lives in the drawer test.
  test("multi mode wires a server-querying onSearch (guard)", async () => {
    const { unmount } = render(
      <DatabaseSelect
        multiple
        value={[]}
        onChange={vi.fn()}
        projectName="projects/p"
      />
    );
    await flush();

    expect(combo.props?.onSearch).toBeTypeOf("function");
    act(() => combo.props?.onSearch?.("orders"));
    expect(mocks.fetchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "projects/p",
        filter: expect.objectContaining({ query: "orders" }),
      })
    );
    unmount();
  });

  test("a selected value absent from current results still renders as an option (chip preservation)", async () => {
    // Server never returns the pre-selected db (it's past page 1).
    mocks.fetchDatabases.mockResolvedValue({
      databases: [db("instances/i/databases/other")],
      nextPageToken: "",
    });
    const { unmount } = render(
      <DatabaseSelect
        multiple
        value={["instances/i/databases/mydb"]}
        onChange={vi.fn()}
        projectName="projects/p"
      />
    );
    await flush();

    const opt = combo.props?.options.find(
      (o) => o.value === "instances/i/databases/mydb"
    );
    expect(opt).toBeDefined();
    expect(opt?.label).toBe("mydb"); // derived from resource name
    unmount();
  });

  test("dedup: a value in BOTH results and selection yields exactly one option", async () => {
    mocks.fetchDatabases.mockResolvedValue({
      databases: [db("instances/i/databases/dup")],
      nextPageToken: "",
    });
    const { unmount } = render(
      <DatabaseSelect
        multiple
        value={["instances/i/databases/dup"]}
        onChange={vi.fn()}
        projectName="projects/p"
      />
    );
    await flush();

    const matches = combo.props?.options.filter(
      (o) => o.value === "instances/i/databases/dup"
    );
    expect(matches).toHaveLength(1);
    unmount();
  });

  test("out-of-order fetch responses do not clobber the latest results (race guard)", async () => {
    let resolveMount: (v: unknown) => void = () => {};
    let resolveSearch: (v: unknown) => void = () => {};
    mocks.fetchDatabases
      .mockImplementationOnce(
        () =>
          new Promise((r) => {
            resolveMount = r;
          })
      )
      .mockImplementationOnce(
        () =>
          new Promise((r) => {
            resolveSearch = r;
          })
      );

    const { unmount } = render(
      <DatabaseSelect
        multiple
        value={[]}
        onChange={vi.fn()}
        projectName="projects/p"
      />
    );
    // Mount fetch (call 1) is in flight; fire a search (call 2).
    act(() => combo.props?.onSearch?.("q"));

    // Resolve the NEWER search first, then the OLDER mount fetch.
    await act(async () => {
      resolveSearch({
        databases: [db("instances/i/databases/newer")],
        nextPageToken: "",
      });
      await Promise.resolve();
    });
    await act(async () => {
      resolveMount({
        databases: [db("instances/i/databases/older")],
        nextPageToken: "",
      });
      await Promise.resolve();
    });

    const values = combo.props?.options.map((o) => o.value) ?? [];
    expect(values).toContain("instances/i/databases/newer");
    expect(values).not.toContain("instances/i/databases/older");
    unmount();
  });
});

describe("DatabaseSelect — single mode (regression guard for Sync Schema)", () => {
  test("onChange yields the real Database (not a synthesized stub)", async () => {
    const dbS = db("instances/i/databases/s", "environments/staging");
    mocks.cache.set(dbS.name, dbS);
    mocks.fetchDatabases.mockResolvedValue({
      databases: [dbS],
      nextPageToken: "",
    });
    const onChange = vi.fn();
    const { unmount } = render(
      <DatabaseSelect value="" onChange={onChange} projectName="projects/p" />
    );
    await flush();

    act(() => combo.props?.onChange("instances/i/databases/s"));
    expect(onChange).toHaveBeenCalledWith(
      "instances/i/databases/s",
      expect.objectContaining({ effectiveEnvironment: "environments/staging" })
    );
    unmount();
  });

  test("selected label survives a search that drops it from the page (cache-resolved)", async () => {
    const keep = db("instances/i/databases/keep", "environments/prod");
    mocks.cache.set(keep.name, keep);
    mocks.fetchDatabases.mockResolvedValue({
      databases: [keep],
      nextPageToken: "",
    });
    const { unmount } = render(
      <DatabaseSelect
        value="instances/i/databases/keep"
        onChange={vi.fn()}
        projectName="projects/p"
      />
    );
    await flush();

    // A search returns a different page not containing "keep".
    mocks.fetchDatabases.mockResolvedValue({
      databases: [db("instances/i/databases/zzz")],
      nextPageToken: "",
    });
    act(() => combo.props?.onSearch?.("zzz"));
    await flush();

    // The selected value must still be resolvable as an option/label.
    const opt = combo.props?.options.find(
      (o) => o.value === "instances/i/databases/keep"
    );
    expect(opt).toBeDefined();
    expect(opt?.label).toBe("keep");
    unmount();
  });

  test("resolves a selected off-page value from the store cache (Sync Schema deep-link)", async () => {
    const cached = db("instances/i/databases/deeplink", "environments/prod");
    mocks.cache.set(cached.name, cached);
    // The mount fetch page does NOT contain the pre-selected value.
    mocks.fetchDatabases.mockResolvedValue({
      databases: [db("instances/i/databases/other")],
      nextPageToken: "",
    });
    const onChange = vi.fn();
    const { unmount } = render(
      <DatabaseSelect
        value="instances/i/databases/deeplink"
        onChange={onChange}
        projectName="projects/p"
      />
    );
    await flush();

    // Re-selecting the pre-filled (off-page) value must yield the REAL cached
    // Database, not undefined (undefined would make Sync Schema clear it).
    act(() => combo.props?.onChange("instances/i/databases/deeplink"));
    expect(onChange).toHaveBeenCalledWith(
      "instances/i/databases/deeplink",
      expect.objectContaining({ effectiveEnvironment: "environments/prod" })
    );
    unmount();
  });
});
