import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import type {
  AccessGrantDraft,
  DeniedQueryRequestDraft,
} from "@/modules/sql-editor/model/accessGrantDraft";
import { State } from "@/types/proto-es/v1/common_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { validateAccessGrantDraft } from "./deniedQueryAccess";
import { getMaskingSetup } from "./tools/maskingSetup";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);
const isProject = (value: unknown): value is string =>
  typeof value === "string" && /^projects\/(?!-$)[A-Za-z0-9_-]+$/.test(value);
const isDatabase = (value: unknown): value is string =>
  typeof value === "string" &&
  /^(?:projects\/[A-Za-z0-9_-]+\/)?instances\/[^/]+\/databases\/[^/]+$/.test(
    value
  ) &&
  !value.split("/").includes("-");

export function withPreparedAccessGrantReason(
  draft: DeniedQueryRequestDraft | undefined,
  prepared: AccessGrantDraft | undefined
): DeniedQueryRequestDraft | undefined {
  if (!draft || draft.kind === "requestRole" || prepared?.reason === undefined)
    return draft;
  if (
    draft.project !== prepared.project ||
    draft.sourceDatabase !== prepared.sourceDatabase ||
    draft.statement !== prepared.statement ||
    (draft.schema ?? "") !== (prepared.schema ?? "") ||
    (draft.container ?? "") !== (prepared.container ?? "")
  )
    return draft;
  const targets = new Set(draft.targets);
  const preparedTargets = new Set(prepared.targets);
  if (
    targets.size !== preparedTargets.size ||
    ![...targets].every((target) => preparedTargets.has(target))
  )
    return draft;
  return { ...draft, reason: prepared.reason };
}

function readDraft(value: unknown): AccessGrantDraft | undefined {
  if (!isRecord(value)) return undefined;
  const {
    project,
    sourceDatabase,
    statement,
    unmask,
    schema,
    container,
    reason,
  } = value;
  const targets = value.targets;
  if (
    !isProject(project) ||
    !isDatabase(sourceDatabase) ||
    typeof statement !== "string" ||
    !statement.trim() ||
    statement.length > 100_000 ||
    typeof unmask !== "boolean" ||
    !Array.isArray(targets) ||
    !targets.length ||
    !targets.every(isDatabase) ||
    (schema !== undefined && typeof schema !== "string") ||
    (reason !== undefined && typeof reason !== "string") ||
    (container !== undefined && typeof container !== "string")
  )
    return undefined;
  return {
    kind: "accessGrant",
    project,
    sourceDatabase,
    targets: [...new Set(targets)],
    statement,
    unmask,
    ...(reason !== undefined ? { reason } : {}),
    ...(schema ? { schema } : {}),
    ...(container ? { container } : {}),
  };
}

export function parsePreparedAccessGrantDraft(
  result: string
): AccessGrantDraft | undefined {
  try {
    const value: unknown = JSON.parse(result);
    return isRecord(value) &&
      value.status === "unsaved" &&
      !value.error &&
      isRecord(value.draft) &&
      value.draft.kind === "accessGrant"
      ? readDraft(value.draft)
      : undefined;
  } catch {
    return undefined;
  }
}

export function createJustInTimeTools(options: {
  workspaceName?: string;
  projectName?: string;
  databaseName?: string;
}) {
  const boundProject = isProject(options.projectName)
    ? options.projectName
    : undefined;
  const error = (message: string) => JSON.stringify({ error: message });
  return {
    async overview(args: Record<string, unknown>): Promise<string> {
      if (!options.workspaceName)
        return error("A signed-in workspace is required.");
      let project = args.project || boundProject;
      const databaseName = args.database || options.databaseName;
      if (
        (project && !isProject(project)) ||
        (boundProject && project !== boundProject)
      )
        return error("Select an exact project within this chat's scope.");
      if (
        databaseName &&
        (!isDatabase(databaseName) ||
          (options.databaseName && databaseName !== options.databaseName))
      )
        return error("Select an exact database within this turn's scope.");
      try {
        const database = databaseName
          ? await databaseServiceClientConnect.getDatabase(
              create(GetDatabaseRequestSchema, {
                name: databaseName as string,
              }),
              {
                contextValues: createContextValues().set(
                  silentContextKey,
                  true
                ),
              }
            )
          : undefined;
        if (database) {
          if (
            database.name !== databaseName ||
            database.state !== State.ACTIVE ||
            !isProject(database.project) ||
            (project && project !== database.project)
          )
            return error(
              "The database is unavailable in the selected project."
            );
          project = database.project;
        }
        const setup = await getMaskingSetup({
          workspaceName: options.workspaceName,
          projectName: project as string | undefined,
          databaseName: database?.name,
          database,
        });
        return JSON.stringify({
          ...(project ? { project } : {}),
          ...(database ? { database: database.name } : {}),
          justInTime: setup.justInTime,
          subscription: setup.subscription,
          databasePicker: setup.scopeAccess?.databasePicker,
        });
      } catch {
        return error(
          "Unable to verify current Just-in-Time access eligibility."
        );
      }
    },
    async prepare(args: Record<string, unknown>): Promise<string> {
      if (!options.workspaceName)
        return error("A signed-in workspace is required.");
      const draft = readDraft({
        ...args,
        sourceDatabase: args.database,
        targets: args.targets ?? [args.database],
      });
      if (!draft)
        return error(
          "Provide an exact project, database, SQL statement (at most 100000 characters), and explicit Unmask choice."
        );
      if (
        (boundProject && draft.project !== boundProject) ||
        (options.databaseName && draft.sourceDatabase !== options.databaseName)
      )
        return error(
          "The request must stay within this chat's selected resource scope."
        );
      if (!(await validateAccessGrantDraft(draft)))
        return error(
          "The JIT request is not currently available for this member and target. Refresh get_just_in_time_overview for current eligibility."
        );
      return JSON.stringify({ status: "unsaved", draft });
    },
  };
}
