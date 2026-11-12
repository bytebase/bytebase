import { RULE_REVIEW_ENGINES } from "@/components/sql-review/standardRuleCatalog";
import { isValidDatabaseGroupName } from "@/types/dbGroup";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";

const RULE_REVIEW_ENGINE_SET: ReadonlySet<Engine> = new Set(
  RULE_REVIEW_ENGINES
);

export interface ReviewScope {
  // The databases the review reads, in plan order.
  databases: string[];
  engines: Engine[];
  // The databases it could have read; more than `databases` when the project
  // sets a sample size.
  total: number;
}

// What the review covers. It takes the first `sampleSize` targets of each
// spec, or every target when the size is zero. Of those the AI review reads
// all, and the standard rules read the ones on an engine they cover. A
// group's targets are its matched databases once loaded.
export function reviewScope(
  plan: Pick<Plan, "specs">,
  options: {
    reviewers: { rules: boolean; ai: boolean };
    sampleSize: number;
    matchedDatabases: (group: string) => string[] | undefined;
    engineOf: (database: string) => Engine | undefined;
  }
): ReviewScope {
  const { reviewers } = options;
  const read = (database: string) => {
    if (reviewers.ai) return true;
    if (!reviewers.rules) return false;
    const engine = options.engineOf(database);
    return engine === undefined || RULE_REVIEW_ENGINE_SET.has(engine);
  };
  const reviewed = new Set<string>();
  const candidates = new Set<string>();
  for (const spec of plan.specs) {
    if (
      spec.config.case !== "changeDatabaseConfig" ||
      spec.config.value.release
    ) {
      continue;
    }
    const targets = spec.config.value.targets.flatMap((target) =>
      isValidDatabaseGroupName(target)
        ? (options.matchedDatabases(target) ?? [])
        : [target]
    );
    const sampled =
      options.sampleSize > 0 ? targets.slice(0, options.sampleSize) : targets;
    for (const database of targets.filter(read)) candidates.add(database);
    for (const database of sampled.filter(read)) reviewed.add(database);
  }
  const databases = [...reviewed];
  const engines = [
    ...new Set(
      databases
        .map((database) => options.engineOf(database))
        .filter((engine): engine is Engine => engine !== undefined)
    ),
  ];
  return { databases, engines, total: candidates.size };
}
