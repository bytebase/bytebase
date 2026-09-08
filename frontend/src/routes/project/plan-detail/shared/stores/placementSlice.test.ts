import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, test, vi } from "vitest";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  IssueCommentSchema,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { type Sheet, SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { PLACEMENT_BUDGETS } from "../../components/threads/placement/budgets";
import {
  current,
  diffPair,
  tokenizeSheet,
  UNAVAILABLE,
} from "../../components/threads/placement/place";
import {
  type PlacementClient,
  PlacementError,
} from "../../components/threads/placement/placementClient";
import {
  type PlacementRequest,
  type PlacementResponse,
  runPlacementBatch,
} from "../../components/threads/placement/placementWorker";
import type { PlacementDeps } from "./placementSlice";
import { createPlanDetailStore } from "./usePlanDetailStore";

const PROJECT = "projects/p";
const SHA_OLD = "a".repeat(64);
const SHA_NEW = "b".repeat(64);
const SHA_NEWER = "c".repeat(64);
const sheetName = (sha: string) => `${PROJECT}/sheets/${sha}`;
const PREVIEW_BYTES = 4;

const OLD_TEXT = "CREATE TABLE a (id INT);\nALTER TABLE a ADD b INT;\n";
const NEW_TEXT = `-- header\n${OLD_TEXT}`;
const NEWER_TEXT = `-- one\n-- two\n${OLD_TEXT}`;

const anchor = (sha: string, first: number, last = first, spec = "spec-1") =>
  create(StatementAnchorSchema, {
    spec,
    sheetSha256: sha,
    startPosition: create(PositionSchema, { line: first, column: 0 }),
    endPosition: create(PositionSchema, { line: last, column: 0 }),
  });

const nameOf = (id: string) => `${PROJECT}/issues/1/issueComments/${id}`;
const comment = (
  id: string,
  statementAnchor?: ReturnType<typeof anchor>,
  createdAt = 1
) =>
  create(IssueCommentSchema, {
    name: nameOf(id),
    createTime: create(TimestampSchema, { seconds: BigInt(createdAt) }),
    statementAnchor,
  });

const specs = (sha: string, specId = "spec-1") => [
  create(Plan_SpecSchema, {
    id: specId,
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, {
        sheet: sheetName(sha),
      }),
    },
  }),
];

// A sheet cache that behaves like the app store's: previews are truncated to
// PREVIEW_BYTES with the full size attached, raw fetches are complete, and a
// complete sheet is never replaced by a preview.
const fakeSheets = (
  contents: Record<string, string>,
  previewBytes = PREVIEW_BYTES
) => {
  const cache = new Map<string, Sheet>();
  const calls: { name: string; raw: boolean }[] = [];
  const encoder = new TextEncoder();
  const build = (name: string, raw: boolean) => {
    const full = encoder.encode(contents[name]);
    return create(SheetSchema, {
      name,
      content: raw ? full : full.slice(0, previewBytes),
      contentSize: BigInt(full.byteLength),
    });
  };
  return {
    cache,
    calls,
    getSheet: (name: string) => cache.get(name),
    fetchSheet: async (name: string, raw: boolean) => {
      calls.push({ name, raw });
      if (!(name in contents)) return undefined;
      const sheet = build(name, raw);
      const existing = cache.get(name);
      if (!existing || raw) cache.set(name, sheet);
      return cache.get(name);
    },
    prime: (name: string, raw: boolean) => {
      cache.set(name, build(name, raw));
    },
  };
};

const immediateClient = (): PlacementClient & {
  requests: PlacementRequest[];
} => {
  const requests: PlacementRequest[] = [];
  return {
    requests,
    compute: async (request) => {
      requests.push(request);
      return runPlacementBatch(request);
    },
    cancel: vi.fn(),
  };
};

// A client whose answer the test releases by hand. Like the real client,
// `cancel` rejects the request in flight.
const heldClient = () => {
  let release: ((response: PlacementResponse) => void) | undefined;
  let fail: ((error: unknown) => void) | undefined;
  const requests: PlacementRequest[] = [];
  const client: PlacementClient = {
    compute: (request) => {
      requests.push(request);
      return new Promise<PlacementResponse>((resolve, reject) => {
        release = resolve;
        fail = reject;
      });
    },
    cancel: vi.fn(() => fail?.(new PlacementError("cancelled"))),
  };
  return {
    client,
    requests,
    release: () => release?.(runPlacementBatch(requests[requests.length - 1])),
    fail: (error: unknown) => fail?.(error),
  };
};

const setup = (
  overrides: Partial<PlacementDeps> & {
    sheets?: ReturnType<typeof fakeSheets>;
  } = {}
) => {
  const sheets =
    overrides.sheets ??
    fakeSheets({
      [sheetName(SHA_OLD)]: OLD_TEXT,
      [sheetName(SHA_NEW)]: NEW_TEXT,
      [sheetName(SHA_NEWER)]: NEWER_TEXT,
    });
  const client = overrides.client ?? immediateClient();
  let clock = 0;
  const store = createPlanDetailStore({
    getSheet: sheets.getSheet,
    fetchSheet: sheets.fetchSheet,
    client,
    budgets: PLACEMENT_BUDGETS,
    now: () => (clock += 5),
    ...overrides,
  });
  return { store, sheets, client };
};

const compute = (
  store: ReturnType<typeof createPlanDetailStore>,
  comments: ReturnType<typeof comment>[],
  targetSha = SHA_NEW
) =>
  store.getState().computePlacements({
    comments,
    projectName: PROJECT,
    specs: specs(targetSha),
  });

describe("placementSlice", () => {
  test("settles hash-equal anchors without fetching or diffing", async () => {
    const { store, sheets, client } = setup();
    await compute(store, [
      comment("same", anchor(SHA_NEW, 2, 3)),
      comment("plain"),
    ]);
    const state = store.getState();
    expect(state.placements.get(nameOf("same"))).toEqual(current(2, 3));
    expect(state.placements.has(nameOf("plain"))).toBe(false);
    expect(sheets.calls).toEqual([]);
    expect((client as ReturnType<typeof immediateClient>).requests).toEqual([]);
    expect(state.placementMetrics).toMatchObject({
      pairCount: 0,
      computedPairCount: 0,
      sheetCount: 0,
      bytes: 0,
      work: 0,
    });
  });

  test("probes unknown sizes, downloads complete sheets, and places through the worker", async () => {
    const { store, sheets, client } = setup();
    // The current sheet is cached as a preview, like the editor leaves it.
    sheets.prime(sheetName(SHA_NEW), false);
    await compute(store, [
      comment("moved", anchor(SHA_OLD, 1, 2), 1),
      comment("also", anchor(SHA_OLD, 2), 2),
    ]);
    const state = store.getState();
    expect(state.placements.get(nameOf("moved"))).toEqual(current(2, 3));
    expect(state.placements.get(nameOf("also"))).toEqual(current(3, 3));
    expect(sheets.calls).toEqual([
      // Size probe for the never-seen source, then raw downloads of both.
      { name: sheetName(SHA_OLD), raw: false },
      { name: sheetName(SHA_OLD), raw: true },
      { name: sheetName(SHA_NEW), raw: true },
    ]);
    const requests = (client as ReturnType<typeof immediateClient>).requests;
    expect(requests).toHaveLength(1);
    expect(requests[0].sheets).toEqual({
      [sheetName(SHA_OLD)]: OLD_TEXT,
      [sheetName(SHA_NEW)]: NEW_TEXT,
    });
    expect(requests[0].pairs).toEqual([
      expect.objectContaining({
        saved: sheetName(SHA_OLD),
        current: sheetName(SHA_NEW),
        anchors: [
          { id: "1-2", startLine: 1, endLine: 2 },
          { id: "2-2", startLine: 2, endLine: 2 },
        ],
      }),
    ]);
    expect(state.placementMetrics).toMatchObject({
      pairCount: 1,
      computedPairCount: 1,
      sheetCount: 2,
      bytes: OLD_TEXT.length + NEW_TEXT.length,
    });
    expect(state.placementMetrics?.work).toBeGreaterThan(0);
    expect(state.placementMetrics?.durationMs).toBeGreaterThan(0);
  });

  test("reuses a diffed pair on the next run without fetching or diffing again", async () => {
    const { store, sheets, client } = setup();
    await compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
    const callsAfterFirst = sheets.calls.length;
    await compute(store, [
      comment("moved", anchor(SHA_OLD, 1)),
      comment("new-anchor", anchor(SHA_OLD, 2)),
    ]);
    const state = store.getState();
    expect(state.placements.get(nameOf("moved"))).toEqual(current(2, 2));
    // A new range on a cached pair is computed, but the sheets are complete.
    expect(state.placements.get(nameOf("new-anchor"))).toEqual(current(3, 3));
    expect(sheets.calls).toHaveLength(callsAfterFirst);
    expect(
      (client as ReturnType<typeof immediateClient>).requests
    ).toHaveLength(2);
    expect(state.placementMetrics).toMatchObject({
      pairCount: 1,
      computedPairCount: 1,
      sheetCount: 0,
    });
  });

  test("a range already computed on the cached pair costs no worker call", async () => {
    const { store, client } = setup();
    await compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
    await compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
    expect(
      (client as ReturnType<typeof immediateClient>).requests
    ).toHaveLength(1);
    expect(store.getState().placementMetrics).toMatchObject({
      pairCount: 1,
      computedPairCount: 0,
    });
  });

  test("settles UNAVAILABLE when the worker fails or times out", async () => {
    const held = heldClient();
    const { store } = setup({ client: held.client });
    const run = compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
    await vi.waitFor(() => expect(held.requests).toHaveLength(1));
    held.fail(new PlacementError("deadline"));
    await run;
    expect(store.getState().placements.get(nameOf("moved"))).toEqual(
      UNAVAILABLE
    );
    expect(store.getState().placementMetrics).toMatchObject({
      computedPairCount: 1,
      work: 0,
    });
  });

  test("settles UNAVAILABLE when a sheet cannot be downloaded", async () => {
    const sheets = fakeSheets({ [sheetName(SHA_NEW)]: NEW_TEXT });
    const { store, client } = setup({ sheets });
    await compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
    expect(store.getState().placements.get(nameOf("moved"))).toEqual(
      UNAVAILABLE
    );
    expect((client as ReturnType<typeof immediateClient>).requests).toEqual([]);
  });

  test("settles UNAVAILABLE when a raw download comes back incomplete", async () => {
    const sheets = fakeSheets({
      [sheetName(SHA_OLD)]: OLD_TEXT,
      [sheetName(SHA_NEW)]: NEW_TEXT,
    });
    const truncating = {
      ...sheets,
      fetchSheet: async (name: string, raw: boolean) => {
        const sheet = await sheets.fetchSheet(name, raw);
        if (sheet && raw) {
          sheets.cache.set(
            name,
            create(SheetSchema, {
              ...sheet,
              content: sheet.content.slice(0, 2),
            })
          );
        }
        return sheets.cache.get(name);
      },
    };
    const { store, client } = setup({ sheets: truncating });
    await compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
    expect(store.getState().placements.get(nameOf("moved"))).toEqual(
      UNAVAILABLE
    );
    expect((client as ReturnType<typeof immediateClient>).requests).toEqual([]);
  });

  test("a save during the diff supersedes the run and diffs against the new target", async () => {
    const held = heldClient();
    const { store } = setup({ client: held.client });
    const first = compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
    await vi.waitFor(() => expect(held.requests).toHaveLength(1));
    // The author saves a new version while the diff is in flight.
    const next = compute(
      store,
      [comment("moved", anchor(SHA_OLD, 1))],
      SHA_NEWER
    );
    await vi.waitFor(() => expect(held.requests).toHaveLength(2));
    expect(held.requests[1].sheets[sheetName(SHA_NEWER)]).toBe(NEWER_TEXT);
    // The late answer for the old target is dropped.
    held.release();
    await Promise.all([first, next]);
    expect(store.getState().placements.get(nameOf("moved"))).toEqual(
      current(3, 3)
    );
  });

  test("a newer run supersedes an older one and its late result is dropped", async () => {
    const held = heldClient();
    const { store } = setup({ client: held.client });
    const first = compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
    await vi.waitFor(() => expect(held.requests).toHaveLength(1));
    const second = compute(store, [comment("same", anchor(SHA_NEW, 1))]);
    expect(held.client.cancel).toHaveBeenCalled();
    held.release();
    await Promise.all([first, second]);
    const state = store.getState();
    expect(state.placements.has(nameOf("moved"))).toBe(false);
    expect(state.placements.get(nameOf("same"))).toEqual(current(1, 1));
  });

  test("publishes hash-settled placements before the diff finishes", async () => {
    const held = heldClient();
    const { store } = setup({ client: held.client });
    const run = compute(store, [
      comment("same", anchor(SHA_NEW, 1)),
      comment("moved", anchor(SHA_OLD, 1)),
    ]);
    await vi.waitFor(() => expect(held.requests).toHaveLength(1));
    expect(store.getState().placements.get(nameOf("same"))).toEqual(
      current(1, 1)
    );
    expect(store.getState().placements.has(nameOf("moved"))).toBe(false);
    held.release();
    await run;
    expect(store.getState().placements.get(nameOf("moved"))).toEqual(
      current(2, 2)
    );
  });

  test("passes the fixed budgets to the worker", async () => {
    const { store, client } = setup();
    await compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
    const request = (client as ReturnType<typeof immediateClient>).requests[0];
    expect(request.limits).toBe(PLACEMENT_BUDGETS);
    expect(request.generation).toBe(1);
  });

  test("bounds the size probe by the sheet cap", async () => {
    const contents: Record<string, string> = {
      [sheetName(SHA_NEW)]: NEW_TEXT,
    };
    const shas = Array.from({ length: 6 }, (_, i) => String(i + 1).repeat(64));
    for (const sha of shas) contents[sheetName(sha)] = OLD_TEXT;
    const sheets = fakeSheets(contents);
    const { store } = setup({
      sheets,
      budgets: { ...PLACEMENT_BUDGETS, maxSheets: 3 },
    });
    await compute(
      store,
      shas.map((sha, i) => comment(`c${i}`, anchor(sha, 1), i))
    );
    const probes = sheets.calls.filter((call) => !call.raw);
    // Only three sizes are learned: the first source, the shared target, and
    // the second source. Pairs whose source size stays unknown are never
    // downloaded.
    expect(probes.map((call) => call.name)).toEqual([
      sheetName(shas[0]),
      sheetName(SHA_NEW),
      sheetName(shas[1]),
    ]);
    const placed = shas.map((_, i) =>
      store.getState().placements.get(nameOf(`c${i}`))
    );
    expect(placed.slice(0, 2)).toEqual([current(2, 2), current(2, 2)]);
    expect(placed.slice(2)).toEqual(shas.slice(2).map(() => UNAVAILABLE));
    expect(sheets.calls.filter((call) => call.raw)).toHaveLength(3);
  });
});

test("invalidates changed targets before awaiting previews, preserving unchanged specs", async () => {
  const sheets = fakeSheets({
    [sheetName(SHA_OLD)]: OLD_TEXT,
    [sheetName(SHA_NEW)]: NEW_TEXT,
    [sheetName(SHA_NEWER)]: NEWER_TEXT,
  });
  let release!: () => void;
  const heldSheets = {
    ...sheets,
    fetchSheet: async (name: string, raw: boolean) => {
      if (name === sheetName(SHA_NEWER) && !raw)
        await new Promise<void>((resolve) => {
          release = resolve;
        });
      return sheets.fetchSheet(name, raw);
    },
  };
  const { store } = setup({ sheets: heldSheets });
  const comments = [
    comment("moved", anchor(SHA_OLD, 1)),
    comment("unchanged", anchor(SHA_NEW, 1, 1, "spec-2")),
  ];
  const stableSpec = specs(SHA_NEW, "spec-2");
  await store.getState().computePlacements({
    comments,
    projectName: PROJECT,
    specs: [...specs(SHA_NEW), ...stableSpec],
  });
  expect(store.getState().placements.get(nameOf("moved"))).toEqual(
    current(2, 2)
  );
  const pending = store.getState().computePlacements({
    comments,
    projectName: PROJECT,
    specs: [...specs(SHA_NEWER), ...stableSpec],
  });
  const duringProbe = new Map(store.getState().placements);
  release();
  await pending;
  expect(duringProbe.has(nameOf("moved"))).toBe(false);
  expect(duringProbe.get(nameOf("unchanged"))).toEqual(current(1, 1));
  expect(store.getState().placements.get(nameOf("moved"))).toEqual(
    current(3, 3)
  );
});

test("caches a definitive UNAVAILABLE so the pair is not diffed again", async () => {
  const { store, client } = setup({
    budgets: { ...PLACEMENT_BUDGETS, maxLinesPerSheet: 1 },
  });
  const input = {
    comments: [comment("moved", anchor(SHA_OLD, 1))],
    projectName: PROJECT,
    specs: specs(SHA_NEW),
  };
  await store.getState().computePlacements(input);
  expect(store.getState().placements.get(nameOf("moved"))).toEqual(UNAVAILABLE);
  await store.getState().computePlacements(input);
  const requests = (client as ReturnType<typeof immediateClient>).requests;
  expect(requests).toHaveLength(1);
});

test("reserves the per-sheet cap before each probe so concurrent probes stay within the total", async () => {
  const contents: Record<string, string> = { [sheetName(SHA_NEW)]: NEW_TEXT };
  const shas = Array.from({ length: 4 }, (_, i) => String(i + 1).repeat(64));
  for (const sha of shas) contents[sheetName(sha)] = OLD_TEXT;
  // Like the server at production budgets, a preview of a small sheet is
  // already complete.
  const sheets = fakeSheets(contents, Number.POSITIVE_INFINITY);
  const { store } = setup({
    sheets,
    budgets: {
      ...PLACEMENT_BUDGETS,
      // Two reservations fit; the third would exceed the total, so it never
      // starts even though four probes could run at once.
      maxBytesPerSheet: 100,
      maxTotalBytes: 200,
    },
  });
  await store.getState().computePlacements({
    comments: shas.map((sha, i) => comment(`c${i}`, anchor(sha, 1), i)),
    projectName: PROJECT,
    specs: specs(SHA_NEW),
  });
  expect(
    sheets.calls.filter((call) => !call.raw).map((call) => call.name)
  ).toEqual([sheetName(shas[0]), sheetName(SHA_NEW)]);
  expect(store.getState().placements.get(nameOf("c0"))).toEqual(current(2, 2));
  expect(store.getState().placements.get(nameOf("c1"))).toEqual(UNAVAILABLE);
});

test("counts probe bytes against the raw download budget", async () => {
  // Previews come back truncated here, so the sheets still need a raw fetch
  // that must fit in whatever the probes left of the budget.
  const { store, sheets } = setup({
    budgets: {
      ...PLACEMENT_BUDGETS,
      maxBytesPerSheet: 100,
      maxTotalBytes: 100,
    },
  });
  await compute(store, [comment("moved", anchor(SHA_OLD, 1))]);
  // One probe spends its bytes; the raw downloads no longer fit.
  expect(sheets.calls.filter((call) => call.raw)).toEqual([]);
  expect(store.getState().placements.get(nameOf("moved"))).toEqual(UNAVAILABLE);
});

test("retries shared-budget failures after other pairs have been cached", async () => {
  const work = diffPair(tokenizeSheet(OLD_TEXT), tokenizeSheet(NEW_TEXT), {
    maxLinesPerSheet: 100,
    maxWork: 10000,
  }).work;
  const { store, client } = setup({
    budgets: { ...PLACEMENT_BUDGETS, maxWorkPerRun: work },
  });
  const first = comment("first", anchor(SHA_OLD, 1, 1, "spec-1"), 1);
  const second = comment("second", anchor(SHA_OLD, 1, 1, "spec-2"), 2);
  const input = {
    comments: [first, second],
    projectName: PROJECT,
    specs: [...specs(SHA_NEW), ...specs(SHA_NEW, "spec-2")],
  };
  await store.getState().computePlacements(input);
  expect(store.getState().placements.get(first.name)).toEqual(current(2, 2));
  expect(store.getState().placements.get(second.name)).toEqual(UNAVAILABLE);
  await store.getState().computePlacements(input);
  expect(store.getState().placements.get(second.name)).toEqual(current(2, 2));
  const requests = (client as ReturnType<typeof immediateClient>).requests;
  expect(requests).toHaveLength(2);
  expect(requests[1].pairs).toHaveLength(1);
});

test.each([false, true])(
  "keeps cached ranges while computing a new range (failure=%s)",
  async (fail) => {
    const held = heldClient();
    const { store } = setup({ client: held.client });
    const old = comment("old", anchor(SHA_OLD, 1));
    const first = compute(store, [old]);
    await vi.waitFor(() => expect(held.requests).toHaveLength(1));
    held.release();
    await first;
    const existing = store.getState().placements.get(old.name);
    const added = comment("new", anchor(SHA_OLD, 2));
    const next = compute(store, [old, added]);
    await vi.waitFor(() => expect(held.requests).toHaveLength(2));
    expect(store.getState().placements.get(old.name)).toBe(existing);
    expect(held.requests[1].pairs[0].anchors).toEqual([
      { id: "2-2", startLine: 2, endLine: 2 },
    ]);
    if (fail) held.fail(new PlacementError("deadline"));
    else held.release();
    await next;
    expect(store.getState().placements.get(old.name)).toBe(existing);
    expect(store.getState().placements.get(added.name)).toEqual(
      fail ? UNAVAILABLE : current(3, 3)
    );
  }
);
