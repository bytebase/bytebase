import { useAppStore } from "@/stores/app";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { getSheetStatement, isSheetContentComplete } from "@/utils/v1/sheet";
import {
  PLACEMENT_BUDGETS,
  type PlacementBudgets,
} from "../../components/threads/placement/budgets";
import { runWithConcurrency } from "../../components/threads/placement/concurrency";
import {
  type Placement,
  UNAVAILABLE,
} from "../../components/threads/placement/place";
import {
  createPlacementClient,
  type PlacementClient,
} from "../../components/threads/placement/placementClient";
import {
  type PlacementPair,
  planPlacements,
} from "../../components/threads/placement/placementPlan";
import type { PlacementRequestPair } from "../../components/threads/placement/placementWorker";
import { targetSha256OfSpec } from "../../components/threads/threadModel";
import type { PlacementSlice, PlanDetailSliceCreator } from "./types";

// What the slice needs from outside: the sheet cache, a way to fill it, and
// the worker. Tests substitute fakes.
export interface PlacementDeps {
  getSheet: (name: string) => Sheet | undefined;
  fetchSheet: (name: string, raw: boolean) => Promise<Sheet | undefined>;
  client: PlacementClient;
  budgets: PlacementBudgets;
  now: () => number;
}

const FETCH_CONCURRENCY = 4;

export const defaultPlacementDeps = (): PlacementDeps => ({
  getSheet: (name) => useAppStore.getState().getSheetByName(name),
  fetchSheet: (name, raw) => useAppStore.getState().fetchSheet(name, raw),
  client: createPlacementClient(),
  budgets: PLACEMENT_BUDGETS,
  now: () => performance.now(),
});

const samePlacement = (a: Placement, b: Placement): boolean =>
  a.state === b.state &&
  (a.state !== "CURRENT" ||
    b.state !== "CURRENT" ||
    (a.range.startLine === b.range.startLine &&
      a.range.endLine === b.range.endLine));

// The placements of the latest run apply to a spec only while it still
// points at the sheet that run was computed against; a consumer showing
// another revision falls back to the no-diff rule instead.
export const placementsForSheet = (
  state: Pick<PlacementSlice, "placements" | "placementTargets">,
  specId: string,
  sheetSha256: string | undefined
): ReadonlyMap<string, Placement> | undefined =>
  sheetSha256 !== undefined &&
  state.placementTargets.get(specId) === sheetSha256
    ? state.placements
    : undefined;

export const placementOf = (
  state: Pick<PlacementSlice, "placements" | "placementTargets">,
  commentName: string,
  specId: string,
  sheetSha256: string | undefined
): Placement | undefined =>
  placementsForSheet(state, specId, sheetSha256)?.get(commentName);

// Computes where every anchored comment shows for the plan's current
// statements. Placement is derived, never stored server-side: each run plans
// the sheets it needs within the fetch budget, loads them complete, diffs
// each saved/current pair once in a worker, and applies the result only while
// the run is still the latest. Pair results are cached by content identity,
// so a repeated run for unchanged sheets costs nothing.
export const createPlacementSlice =
  (deps: PlacementDeps): PlanDetailSliceCreator<PlacementSlice> =>
  (set, get) => {
    const pairCache = new Map<string, Record<string, Placement>>();

    // Settles the comments behind each of the pair's anchors; an anchor the
    // results do not cover is UNAVAILABLE.
    const applyPair = (
      into: Map<string, Placement>,
      pair: PlacementPair,
      results: Record<string, Placement>
    ) => {
      for (const anchor of pair.anchors) {
        const placement = results[anchor.id] ?? UNAVAILABLE;
        for (const name of pair.comments.get(anchor.id) ?? []) {
          into.set(name, placement);
        }
      }
    };

    const settleUnavailable = (
      into: Map<string, Placement>,
      pair: PlacementPair
    ) => applyPair(into, pair, {});

    const completeSheet = (name: string): Sheet | undefined => {
      const sheet = deps.getSheet(name);
      return sheet && isSheetContentComplete(sheet) ? sheet : undefined;
    };

    // Writes the run's targets and placements together, reusing previous
    // entry objects where equal, so consumers selecting the map or one entry
    // rerender only on a real change and never see new targets paired with
    // an older run's placements.
    const publish = (
      next: Map<string, Placement>,
      targets?: ReadonlyMap<string, string>
    ) => {
      const previous = get().placements;
      let changed = previous.size !== next.size;
      const merged = new Map<string, Placement>();
      for (const [name, placement] of next) {
        const before = previous.get(name);
        if (before && samePlacement(before, placement)) {
          merged.set(name, before);
        } else {
          merged.set(name, placement);
          changed = true;
        }
      }
      if (targets) {
        set({ placementTargets: targets, placements: merged });
      } else if (changed) {
        set({ placements: merged });
      }
    };

    // Bumped per run; a run that finds itself superseded stops writing.
    let generation = 0;

    return {
      placements: new Map(),
      placementTargets: new Map(),
      placementMetrics: undefined,

      computePlacements: async ({ comments, projectName, specs }) => {
        const run = ++generation;
        deps.client.cancel();
        const started = deps.now();
        const stale = () => generation !== run;
        const { budgets } = deps;

        const planInput = {
          comments,
          specs,
          projectName,
          sizeOf: (name: string) => deps.getSheet(name)?.contentSize,
          isComplete: (name: string) => completeSheet(name) !== undefined,
          budgets,
        };
        const targets = new Map<string, string>();
        for (const spec of specs) {
          const sha256 = targetSha256OfSpec(spec);
          if (sha256) targets.set(spec.id, sha256);
        }

        // Everything that needs no download or diff settles right away.
        // Cached pairs serve the ranges they have seen, and only the missing
        // ranges go to the worker, so known threads stay visible meanwhile.
        let plan = planPlacements(planInput);
        const placements = new Map<string, Placement>();
        let pending: PlacementPair[] = [];
        const settle = (settleUnknownSizes: boolean) => {
          plan = planPlacements({ ...planInput, settleUnknownSizes });
          placements.clear();
          for (const [name, placement] of plan.settled) {
            placements.set(name, placement);
          }
          pending = [];
          for (const pair of plan.pairs) {
            const cached = pairCache.get(pair.key) ?? {};
            const known = pair.anchors.filter((anchor) => anchor.id in cached);
            const missing = pair.anchors.filter(
              (anchor) => !(anchor.id in cached)
            );
            applyPair(placements, { ...pair, anchors: known }, cached);
            if (missing.length > 0) pending.push({ ...pair, anchors: missing });
          }
        };
        settle(false);
        publish(placements, targets);

        // A sheet the cache has never seen has no known size. A preview fetch
        // is capped by the server at the per-sheet budget, so it doubles as
        // the bounded download; the byte budget applies to it as it goes.
        if (plan.unknownSizes.length > 0) {
          let downloaded = 0n;
          const byteCap = BigInt(budgets.maxTotalBytes);
          await runWithConcurrency(
            plan.unknownSizes.slice(0, budgets.maxSheets),
            FETCH_CONCURRENCY,
            async (name) => {
              if (downloaded >= byteCap) return undefined;
              const sheet = await deps.fetchSheet(name, false);
              if (sheet) downloaded += BigInt(sheet.content.byteLength);
              return sheet;
            }
          );
          if (stale()) return;
          settle(true);
          publish(placements);
        }

        // Sheets the probe left incomplete are re-fetched raw and verified.
        const needed = new Set(
          pending.flatMap((pair) => [pair.sourceName, pair.targetName])
        );
        const fetches = plan.fetches.filter((name) => needed.has(name));
        await runWithConcurrency(fetches, FETCH_CONCURRENCY, (name) =>
          deps.fetchSheet(name, true)
        );
        if (stale()) return;

        // Each sheet travels to the worker once, however many pairs use it.
        let bytes = 0;
        const sheets: Record<string, string> = {};
        const include = (sheet: Sheet) => {
          if (sheet.name in sheets) return;
          sheets[sheet.name] = getSheetStatement(sheet);
          bytes += sheet.content.byteLength;
        };
        const requestPairs: PlacementRequestPair[] = [];
        const requested: PlacementPair[] = [];
        for (const pair of pending) {
          const source = completeSheet(pair.sourceName);
          const target = completeSheet(pair.targetName);
          if (!source || !target) {
            settleUnavailable(placements, pair);
            continue;
          }
          include(source);
          include(target);
          requestPairs.push({
            key: pair.key,
            saved: source.name,
            current: target.name,
            anchors: pair.anchors,
          });
          requested.push(pair);
        }

        let work = 0;
        if (requestPairs.length > 0) {
          try {
            const response = await deps.client.compute(
              { generation: run, sheets, pairs: requestPairs, limits: budgets },
              budgets.workerDeadlineMs
            );
            if (stale()) return;
            work = response.work;
            const incomplete = new Set(response.incomplete);
            for (const pair of requested) {
              const results = response.results[pair.key];
              if (!results) {
                settleUnavailable(placements, pair);
                continue;
              }
              // A pair that ran out of budget may place next time; every
              // other result is a function of the two sheets and is final.
              if (!incomplete.has(pair.key)) {
                pairCache.set(pair.key, {
                  ...pairCache.get(pair.key),
                  ...results,
                });
              }
              applyPair(placements, pair, results);
            }
          } catch {
            if (stale()) return;
            for (const pair of requested) settleUnavailable(placements, pair);
          }
        }

        publish(placements);
        set({
          placementMetrics: {
            durationMs: deps.now() - started,
            pairCount: plan.pairs.length,
            computedPairCount: requestPairs.length,
            sheetCount: fetches.length,
            bytes,
            work,
          },
        });
      },
    };
  };
