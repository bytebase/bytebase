import { create } from "@bufbuild/protobuf";
import { head } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { useStorageStore } from "@/store";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractSheetUID, setSheetStatement } from "@/utils";
import { getLocalSheetByName, getNextLocalSheetUID } from "./localSheet";

type PlanTemplate = "bb.plan.change-database";

type InitialSQL = {
  sqlMap?: Record<string, string>;
  sql?: string;
};

const buildSpecForTargets = (
  targets: string[],
  project: Project,
  sheetUID?: string
) => {
  const sheet = `${project.name}/sheets/${sheetUID ?? getNextLocalSheetUID()}`;
  return create(Plan_SpecSchema, {
    id: uuidv4(),
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, {
        enablePriorBackup: false,
        sheet,
        targets,
      }),
    },
  });
};

const targetsForSpec = (spec: Plan_Spec): string[] => {
  if (spec.config.case === "changeDatabaseConfig") {
    return spec.config.value.targets ?? [];
  }
  if (spec.config.case === "exportDataConfig") {
    return spec.config.value.targets ?? [];
  }
  return [];
};

const maybeSetInitialSQLForSpec = (spec: Plan_Spec, initialSQL: InitialSQL) => {
  const sheet =
    spec.config.case === "changeDatabaseConfig" ||
    spec.config.case === "exportDataConfig"
      ? spec.config.value.sheet
      : "";
  if (!sheet || !extractSheetUID(sheet).startsWith("-")) {
    return;
  }
  const target = head(targetsForSpec(spec));
  const sql = initialSQL.sqlMap?.[target || ""] ?? initialSQL.sql ?? "";
  if (!sql) {
    return;
  }
  setSheetStatement(getLocalSheetByName(sheet), sql);
};

const extractInitialSQLFromQuery = async (
  query: Record<string, string>
): Promise<InitialSQL> => {
  const storageStore = useStorageStore();
  if (query.sql) {
    return { sql: query.sql };
  }
  if (query.sqlStorageKey) {
    return {
      sql: (await storageStore.get<string>(query.sqlStorageKey)) || "",
    };
  }
  const sqlMapText = query.sqlMap;
  if (sqlMapText) {
    try {
      const parsed = JSON.parse(sqlMapText);
      if (typeof parsed === "object" && parsed !== null) {
        return { sqlMap: parsed as Record<string, string> };
      }
    } catch {
      // Fallback to single SQL below.
    }
  }
  if (query.sqlMapStorageKey) {
    return {
      sqlMap:
        (await storageStore.get<Record<string, string>>(
          query.sqlMapStorageKey
        )) || {},
    };
  }
  return {};
};

export const createPlanSkeleton = async (
  project: Project,
  query: Record<string, string>
) => {
  const template = query.template as PlanTemplate | undefined;
  if (!template || template !== "bb.plan.change-database") {
    throw new Error(
      "Only change-database plan creation is supported from the plan detail route."
    );
  }

  const databaseNameList = (query.databaseList ?? "").split(",");
  const targets = query.databaseGroupName
    ? [query.databaseGroupName]
    : databaseNameList;
  const initialSQL = await extractInitialSQLFromQuery(query);
  const plan = create(PlanSchema, {
    description: query.description,
    name: `${project.name}/plans/${getNextLocalSheetUID()}`,
    title: query.name,
  });

  if (initialSQL.sqlMap && Object.keys(initialSQL.sqlMap).length > 0) {
    for (const target of targets) {
      const spec = buildSpecForTargets([target], project);
      maybeSetInitialSQLForSpec(spec, initialSQL);
      plan.specs.push(spec);
    }
  } else {
    const spec = buildSpecForTargets(targets, project);
    maybeSetInitialSQLForSpec(spec, initialSQL);
    plan.specs = [spec];
  }

  return plan;
};
