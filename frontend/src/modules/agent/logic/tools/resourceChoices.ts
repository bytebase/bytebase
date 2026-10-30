import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { State } from "@/types/proto-es/v1/common_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { getDatabaseMentionLabel } from "../resourceMention";
import type { AgentAskUserOption } from "../types";

export type ResourceChoiceKind = "database";

const DATABASE_NAME =
  /^(?:projects\/([A-Za-z0-9_-]+)\/)?instances\/[A-Za-z0-9_-]+\/databases\/[^/]+$/;
const PROJECT_NAME = /^projects\/[A-Za-z0-9_-]+$/;
const WORKSPACE_NAME = /^workspaces\/[A-Za-z0-9_-]+$/;

export async function validateResourceChoices(
  kind: ResourceChoiceKind,
  values: string[],
  projectName: string | undefined,
  workspaceName: string | undefined
): Promise<AgentAskUserOption[] | undefined> {
  if (
    !WORKSPACE_NAME.test(workspaceName ?? "") ||
    values.length === 0 ||
    values.length > 5 ||
    values.some((value) => typeof value !== "string")
  ) {
    return undefined;
  }
  const names = [...new Set(values.map((value) => value.trim()))];
  const contextValues = createContextValues().set(silentContextKey, true);

  if (kind !== "database" || !PROJECT_NAME.test(projectName ?? "")) {
    return undefined;
  }
  const inScope = names.filter((name) => {
    const match = DATABASE_NAME.exec(name);
    return match && (!match[1] || `projects/${match[1]}` === projectName);
  });
  const responses = await Promise.allSettled(
    inScope.map((name) =>
      databaseServiceClientConnect.getDatabase(
        create(GetDatabaseRequestSchema, { name }),
        { contextValues }
      )
    )
  );
  const options = responses.flatMap((response, index) => {
    if (response.status !== "fulfilled") return [];
    const database = response.value;
    if (
      database.name !== inScope[index] ||
      database.project !== projectName ||
      database.state !== State.ACTIVE
    ) {
      return [];
    }
    return [{ label: getDatabaseMentionLabel(database), value: database.name }];
  });
  return options.length ? options : undefined;
}
