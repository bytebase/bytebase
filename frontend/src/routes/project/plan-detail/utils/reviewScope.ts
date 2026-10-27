import { isValidDatabaseGroupName } from "@/types/dbGroup";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";

// Engines the standard rule review covers. Mirrors reviewFuncs in
// backend/runner/reviewrun/review_funcs.go; a target on any other engine is
// reviewed by the plan checks' review policy instead.
const RULE_REVIEW_ENGINES: ReadonlySet<Engine> = new Set([
  Engine.POSTGRES,
  Engine.MYSQL,
  Engine.OCEANBASE,
  Engine.MARIADB,
  Engine.TIDB,
  Engine.MSSQL,
  Engine.ORACLE,
  Engine.SNOWFLAKE,
]);

export interface ReviewScope {
  // The databases the reviewer reviews, in plan order.
  databases: string[];
  engines: Engine[];
  // The databases it could have reviewed; more than `databases` when the AI
  // review samples.
  total: number;
}

// What a reviewer reviews. The rule review takes every target on an engine
// it covers. The AI review takes the first `sampleSize` targets of each
// spec, or every target when the size is zero. A group's targets are its
// matched databases once loaded.
export function reviewScope(
  plan: Pick<Plan, "specs">,
  options: {
    reviewer: "rule" | "ai";
    sampleSize: number;
    matchedDatabases: (group: string) => string[] | undefined;
    engineOf: (database: string) => Engine | undefined;
  }
): ReviewScope {
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
    const eligible =
      options.reviewer === "rule"
        ? targets.filter((database) => {
            const engine = options.engineOf(database);
            return engine === undefined || RULE_REVIEW_ENGINES.has(engine);
          })
        : targets;
    for (const database of eligible) candidates.add(database);
    const sampled =
      options.reviewer === "ai" && options.sampleSize > 0
        ? eligible.slice(0, options.sampleSize)
        : eligible;
    for (const database of sampled) reviewed.add(database);
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
