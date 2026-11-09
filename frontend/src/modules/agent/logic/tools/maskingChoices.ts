import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { settingServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import i18n from "@/lib/i18n";
import { useAppStore } from "@/stores/app";
import { GetSettingRequestSchema } from "@/types/proto-es/v1/setting_service_pb";
import { getSemanticTypeListWithBuiltins } from "@/types/semanticTypes";
import type { AgentAskUserOption } from "../types";
import { getMaskingSetup } from "./maskingSetup";
import { validateResourceChoices } from "./resourceChoices";

interface Context {
  workspaceName: string;
  userName?: string;
}
type Access =
  | { status: string; scope?: string; missingPermissions?: string[] }
  | undefined;

function requireAccess(access: Access, step: string): void {
  if (access?.status !== "allowed")
    throw new Error(
      `${step} access is ${access?.status ?? "unknown"} in ${access?.scope ?? "the selected scope"}. Missing permissions: ${access?.missingPermissions?.join(", ") || "unverified"}.`
    );
}

function contextGuard(context: Context): () => void {
  const userName = context.userName ?? useAppStore.getState().currentUser?.name;
  const check = () => {
    const state = useAppStore.getState();
    if (
      !userName ||
      state.currentUser?.name !== userName ||
      state.workspaceResourceName() !== context.workspaceName
    )
      throw new Error("The masking choice context has changed.");
  };
  check();
  return check;
}

export async function getMaskingSemanticTypeChoices(
  context: Context
): Promise<AgentAskUserOption[]> {
  const check = contextGuard(context);
  const setup = await getMaskingSetup({ workspaceName: context.workspaceName });
  check();
  const step = setup.steps.find((item) => item.id === "semantic-types");
  requireAccess(step?.readAccess, "semantic-types");
  if (setup.features.masking !== "available")
    throw new Error("The masking feature is unavailable or unverified.");
  const setting = await settingServiceClientConnect.getSetting(
    create(GetSettingRequestSchema, { name: "settings/SEMANTIC_TYPES" }),
    { contextValues: createContextValues().set(silentContextKey, true) }
  );
  check();
  if (setting.value?.value.case !== "semanticType")
    throw new Error("The semantic types setting is unavailable.");
  const types = getSemanticTypeListWithBuiltins(
    setting.value.value.value.types
  );
  const seen = new Set<string>();
  const choices = types.map((type) => {
    if (!type.id || seen.has(type.id))
      throw new Error("The semantic type IDs are invalid or ambiguous.");
    seen.add(type.id);
    return {
      label: `${type.title || type.id} (${type.id})`,
      value: `semantic-type:${type.id}`,
      ...(type.description ? { description: type.description } : {}),
    };
  });
  if (step?.configure.status === "allowed")
    choices.push({
      label: i18n.t("agent.masking.semantic-type-choice.create"),
      value: "action:create-semantic-type",
      description: i18n.t(
        "agent.masking.semantic-type-choice.create-description"
      ),
    });
  return choices;
}

export async function getMaskingScopeChoices(
  context: Context & {
    kind: "rule" | "exemption";
    projectName?: string;
    database?: string;
  }
): Promise<AgentAskUserOption[]> {
  const check = contextGuard(context);
  const setup = await getMaskingSetup({
    workspaceName: context.workspaceName,
    projectName: context.projectName,
  });
  check();
  const id = context.kind === "rule" ? "global-masking" : "masking-exemptions";
  const step = setup.steps.find((item) => item.id === id);
  requireAccess(step?.readAccess, id);
  requireAccess(step?.configure, id);
  if (setup.features.masking !== "available")
    throw new Error("The masking feature is unavailable or unverified.");
  const projectAllowed =
    context.projectName && setup.scopeAccess?.project.status === "allowed";
  if (context.kind === "exemption" && !projectAllowed)
    throw new Error("Project scope access is unavailable or unverified.");
  const verified =
    context.database && projectAllowed
      ? await validateResourceChoices(
          "database",
          [context.database],
          context.projectName,
          context.workspaceName
        )
      : undefined;
  check();
  const database =
    verified?.length === 1 && verified[0].value === context.database
      ? verified[0]
      : undefined;
  const choices: AgentAskUserOption[] = [];
  if (database)
    choices.push({
      label: i18n.t("agent.masking.exemption-scope.only-database", {
        database: database.label,
      }),
      value: database.value,
    });
  if (projectAllowed)
    choices.push({
      label: i18n.t("agent.masking.exemption-scope.project"),
      value: context.projectName!,
    });
  if (
    context.projectName &&
    setup.scopeAccess?.databasePicker.status === "allowed"
  )
    choices.push({
      label: i18n.t("agent.masking.exemption-scope.another-database"),
      value: "another-database",
    });
  if (context.kind === "rule") {
    choices.push({
      label: i18n.t("agent.masking.rule-scope.workspace"),
      value: "workspace",
    });
    if (
      !context.projectName &&
      setup.scopeAccess?.projects.status === "allowed"
    )
      choices.push({
        label: i18n.t("agent.masking.rule-scope.choose-project"),
        value: "choose-project",
      });
  }
  if (!choices.length)
    throw new Error("No verified masking scope choices are available.");
  return choices;
}
