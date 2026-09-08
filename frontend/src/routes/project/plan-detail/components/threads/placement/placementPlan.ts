import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  compareByCreateTime,
  sheetNameOfSha256,
  targetSha256OfSpec,
} from "../threadModel";
import type { PlacementBudgets } from "./budgets";
import {
  anchorId,
  type PairAnchor,
  type Placement,
  trivialPlacement,
  UNAVAILABLE,
  wholeLineRange,
} from "./place";

// One saved/current sheet pair to diff, with the anchors that map through it.
export interface PlacementPair {
  readonly key: string;
  readonly specId: string;
  readonly sourceSha256: string;
  readonly targetSha256: string;
  readonly sourceName: string;
  readonly targetName: string;
  readonly anchors: readonly PairAnchor[];
  // Comment names by anchor id.
  readonly comments: ReadonlyMap<string, readonly string[]>;
}

export interface PlacementPlan {
  // Comments settled without loading anything.
  readonly settled: ReadonlyMap<string, Placement>;
  // Pairs to compute, earliest comment first, within the fetch budget.
  readonly pairs: readonly PlacementPair[];
  // Sheet names the pairs need that are not complete in the cache, deduped.
  readonly fetches: readonly string[];
  // Sheets a candidate pair needs whose size is unknown. With
  // `settleUnknownSizes` those pairs' comments are UNAVAILABLE; without it
  // they are left out of `settled`, pending a caller that learns the sizes
  // and plans again.
  readonly unknownSizes: readonly string[];
}

export interface PlacementPlanInput {
  readonly comments: readonly IssueComment[];
  readonly specs: readonly Plan_Spec[];
  // "projects/{project}", the issue's project.
  readonly projectName: string;
  // Content size in bytes when known, from a cached sheet or size metadata.
  readonly sizeOf: (sheetName: string) => bigint | undefined;
  // Whether the cache holds the sheet's complete content.
  readonly isComplete: (sheetName: string) => boolean;
  readonly budgets: Pick<
    PlacementBudgets,
    "maxSheets" | "maxBytesPerSheet" | "maxTotalBytes"
  >;
  // Whether a pair blocked only by an unknown size settles UNAVAILABLE
  // (default) or stays out of the plan for a later pass.
  readonly settleUnknownSizes?: boolean;
}

export const pairKey = (
  specId: string,
  sourceSha256: string,
  targetSha256: string
): string => `${specId} ${sourceSha256} ${targetSha256}`;

interface CandidatePair {
  key: string;
  specId: string;
  sourceSha256: string;
  targetSha256: string;
  sourceName: string;
  targetName: string;
  anchors: Map<string, PairAnchor>;
  comments: Map<string, string[]>;
  // Position of the pair's oldest comment, for deterministic budgeting.
  order: number;
}

// Decides, without any fetch, which comments are already placed, which
// saved/current pairs must be diffed, and which sheets must be downloaded to
// do so. Fetching is bounded here: a pair whose sheets have unknown sizes, a
// sheet over the per-sheet cap, or pairs past the sheet-count or total-byte
// caps settle UNAVAILABLE instead of being fetched.
export function planPlacements(input: PlacementPlanInput): PlacementPlan {
  const settled = new Map<string, Placement>();
  const candidates = new Map<string, CandidatePair>();
  const specs = new Map(input.specs.map((spec) => [spec.id, spec]));
  const ordered = [...input.comments].sort(compareByCreateTime);

  ordered.forEach((comment, order) => {
    const anchor = comment.statementAnchor;
    if (!anchor) return;
    const targetSha256 = targetSha256OfSpec(specs.get(anchor.spec));
    const trivial = trivialPlacement(anchor, targetSha256);
    if (trivial) {
      settled.set(comment.name, trivial);
      return;
    }
    // `trivialPlacement` returned undefined, so both are present.
    const range = wholeLineRange(anchor);
    if (!range || !targetSha256) return;
    const key = pairKey(anchor.spec, anchor.sheetSha256, targetSha256);
    let candidate = candidates.get(key);
    if (!candidate) {
      candidate = {
        key,
        specId: anchor.spec,
        sourceSha256: anchor.sheetSha256,
        targetSha256,
        sourceName: sheetNameOfSha256(input.projectName, anchor.sheetSha256),
        targetName: sheetNameOfSha256(input.projectName, targetSha256),
        anchors: new Map(),
        comments: new Map(),
        order,
      };
      candidates.set(key, candidate);
    }
    const id = anchorId(range);
    candidate.anchors.set(id, { id, ...range });
    candidate.comments.set(id, [
      ...(candidate.comments.get(id) ?? []),
      comment.name,
    ]);
  });

  const pairs: PlacementPair[] = [];
  const fetches: string[] = [];
  const unknownSizes = new Set<string>();
  const fetchSet = new Set<string>();
  let totalBytes = 0n;
  const settleUnavailable = (candidate: CandidatePair) => {
    for (const names of candidate.comments.values()) {
      for (const name of names) settled.set(name, UNAVAILABLE);
    }
  };

  const byOrder = [...candidates.values()].sort((a, b) => a.order - b.order);
  for (const candidate of byOrder) {
    const needed =
      candidate.sourceName === candidate.targetName
        ? [candidate.sourceName]
        : [candidate.sourceName, candidate.targetName];
    let unavailable = false;
    let unknown = false;
    let pairBytes = 0n;
    const pairFetches: string[] = [];
    for (const name of needed) {
      const size = input.sizeOf(name);
      if (size === undefined) {
        unknownSizes.add(name);
        unknown = true;
        continue;
      }
      if (size > BigInt(input.budgets.maxBytesPerSheet)) {
        unavailable = true;
        continue;
      }
      if (input.isComplete(name) || fetchSet.has(name)) continue;
      pairFetches.push(name);
      pairBytes += size;
    }
    if (unknown && !unavailable && input.settleUnknownSizes === false) {
      continue;
    }
    if (
      unknown ||
      unavailable ||
      fetchSet.size + pairFetches.length > input.budgets.maxSheets ||
      totalBytes + pairBytes > BigInt(input.budgets.maxTotalBytes)
    ) {
      settleUnavailable(candidate);
      continue;
    }
    for (const name of pairFetches) {
      fetchSet.add(name);
      fetches.push(name);
    }
    totalBytes += pairBytes;
    pairs.push({
      key: candidate.key,
      specId: candidate.specId,
      sourceSha256: candidate.sourceSha256,
      targetSha256: candidate.targetSha256,
      sourceName: candidate.sourceName,
      targetName: candidate.targetName,
      anchors: [...candidate.anchors.values()],
      comments: candidate.comments,
    });
  }

  return { settled, pairs, fetches, unknownSizes: [...unknownSizes] };
}
