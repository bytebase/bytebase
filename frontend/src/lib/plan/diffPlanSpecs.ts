import { clone, equals } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import {
  type Plan_Spec,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  enablePriorBackupOfSpec,
  sheetNameOfSpec,
  targetsOfSpec,
} from "@/utils/v1/issue/plan";

export type SpecDiffEntry =
  | { kind: "added"; spec: Plan_Spec }
  | { kind: "removed"; spec: Plan_Spec }
  | {
      kind: "updated";
      specId: string;
      from: Plan_Spec;
      to: Plan_Spec;
      sheetChanged: boolean;
      targetsChanged: boolean;
      priorBackupChanged: boolean;
      otherChanged: boolean;
    };

// Compares two specs ignoring the fields rendered explicitly elsewhere
// (sheet, targets, enable_prior_backup). Any residual diff surfaces as a
// generic "updated" fallback row.
function otherFieldsDiffer(a: Plan_Spec, b: Plan_Spec): boolean {
  return !equals(
    Plan_SpecSchema,
    normalizeForOtherDiff(a),
    normalizeForOtherDiff(b)
  );
}

function normalizeForOtherDiff(spec: Plan_Spec): Plan_Spec {
  const c = clone(Plan_SpecSchema, spec);
  if (c.config?.case === "changeDatabaseConfig") {
    c.config.value.sheet = "";
    c.config.value.targets = [];
    c.config.value.enablePriorBackup = false;
  } else if (c.config?.case === "exportDataConfig") {
    c.config.value.sheet = "";
    c.config.value.targets = [];
  }
  return c;
}

const planUpdateDiffCache = new WeakMap<object, SpecDiffEntry[]>();

// Returns the cached diff for a `planUpdate` event payload — the icon and the
// body both need it, so caching by object identity avoids the double pass.
export function diffPlanSpecsForEvent(eventValue: {
  fromSpecs: Plan_Spec[];
  toSpecs: Plan_Spec[];
}): SpecDiffEntry[] {
  const cached = planUpdateDiffCache.get(eventValue);
  if (cached) return cached;
  const fresh = diffPlanSpecs(eventValue.fromSpecs, eventValue.toSpecs);
  planUpdateDiffCache.set(eventValue, fresh);
  return fresh;
}

export function diffPlanSpecs(
  from: Plan_Spec[],
  to: Plan_Spec[]
): SpecDiffEntry[] {
  const fromById = new Map(from.map((s) => [s.id, s]));
  const toById = new Map(to.map((s) => [s.id, s]));
  const out: SpecDiffEntry[] = [];

  for (const s of to) {
    if (!fromById.has(s.id)) out.push({ kind: "added", spec: s });
  }
  for (const s of from) {
    if (!toById.has(s.id)) out.push({ kind: "removed", spec: s });
  }
  for (const newSpec of to) {
    const oldSpec = fromById.get(newSpec.id);
    if (!oldSpec) continue;
    const sheetChanged = sheetNameOfSpec(oldSpec) !== sheetNameOfSpec(newSpec);
    const targetsChanged = !isEqual(
      targetsOfSpec(oldSpec),
      targetsOfSpec(newSpec)
    );
    const priorBackupChanged =
      enablePriorBackupOfSpec(oldSpec) !== enablePriorBackupOfSpec(newSpec);
    const otherChanged = otherFieldsDiffer(oldSpec, newSpec);
    if (sheetChanged || targetsChanged || priorBackupChanged || otherChanged) {
      out.push({
        kind: "updated",
        specId: newSpec.id,
        from: oldSpec,
        to: newSpec,
        sheetChanged,
        targetsChanged,
        priorBackupChanged,
        otherChanged,
      });
    }
  }
  return out;
}

export function diffEntryKey(entry: SpecDiffEntry): string {
  switch (entry.kind) {
    case "added":
      return `add:${entry.spec.id}`;
    case "removed":
      return `rm:${entry.spec.id}`;
    case "updated":
      return `up:${entry.specId}`;
  }
}
