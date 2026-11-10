import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  databaseServiceClientConnect,
  projectServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { buildPlanCreateRoute } from "@/app/router/routeHelpers";
import { saveInitialSQL } from "@/lib/plan/initialSQLStorage";
import { applyPlanTitleToQuery } from "@/lib/plan/title";
import { State } from "@/types/proto-es/v1/common_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { GetProjectRequestSchema } from "@/types/proto-es/v1/project_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import { extractProjectResourceName } from "@/utils/v1/project";
import { extractPlanDraftSQL } from "./responseActions";
import type { AgentResponseAction } from "./types";

type PlanDraftAction = Extract<AgentResponseAction, { kind: "draft-plan" }>;

export class PlanDraftStorageError extends Error {
  constructor() {
    super("Unable to temporarily store the Plan SQL");
    this.name = "PlanDraftStorageError";
  }
}

export async function prepareAgentPlanDraftRoute(
  action: PlanDraftAction,
  answer: string,
  defaultTitle: string
) {
  const sql = extractPlanDraftSQL(answer);
  if (!sql) throw new Error("The answer has no unambiguous SQL draft");

  const contextValues = createContextValues().set(silentContextKey, true);
  const database = await databaseServiceClientConnect.getDatabase(
    create(GetDatabaseRequestSchema, { name: action.database }),
    { contextValues }
  );
  if (
    database.name !== action.database ||
    database.project !== action.project ||
    database.state !== State.ACTIVE
  ) {
    throw new Error("The selected database is no longer in this project");
  }

  const project = await projectServiceClientConnect.getProject(
    create(GetProjectRequestSchema, { name: action.project }),
    { contextValues }
  );
  if (project.name !== action.project || project.state !== State.ACTIVE) {
    throw new Error("The selected project is no longer available");
  }

  const { databaseName } = extractDatabaseResourceName(database.name);
  const query: Record<string, string> = { databaseList: database.name };
  applyPlanTitleToQuery(
    query,
    project,
    () => `[${databaseName}] ${defaultTitle}`
  );
  try {
    query.sqlStorageKey = saveInitialSQL(sql);
  } catch {
    throw new PlanDraftStorageError();
  }
  return buildPlanCreateRoute(extractProjectResourceName(project.name), query);
}
