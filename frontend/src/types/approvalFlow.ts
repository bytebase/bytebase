import { t } from "@/plugins/i18n";
import { displayRoleTitle } from "@/utils/role";
import { PresetRoleType } from "./iam";

export type BuiltinApprovalFlow = {
  id: string;
  title: string;
  description: string;
  roles: string[];
  readonly: true;
};

type BuiltinApprovalFlowDefinition = {
  id: string;
  descriptionKey: string;
  roles: string[];
};

// Built-in approval flow definitions
// Titles and descriptions use i18n keys for localization
const BUILTIN_FLOW_DEFINITIONS: readonly BuiltinApprovalFlowDefinition[] = [
  {
    id: "bb.project-owner",
    descriptionKey:
      "dynamic.custom-approval.approval-flow-builtin.project-owner",
    roles: [PresetRoleType.PROJECT_OWNER],
  },
  {
    id: "bb.workspace-dba",
    descriptionKey:
      "dynamic.custom-approval.approval-flow-builtin.workspace-dba",
    roles: [PresetRoleType.WORKSPACE_DBA],
  },
  {
    id: "bb.workspace-admin",
    descriptionKey:
      "dynamic.custom-approval.approval-flow-builtin.workspace-admin",
    roles: [PresetRoleType.WORKSPACE_ADMIN],
  },
  {
    id: "bb.project-owner-workspace-dba",
    descriptionKey:
      "dynamic.custom-approval.approval-flow-builtin.project-owner-workspace-dba",
    roles: [PresetRoleType.PROJECT_OWNER, PresetRoleType.WORKSPACE_DBA],
  },
  {
    id: "bb.project-owner-workspace-dba-workspace-admin",
    descriptionKey:
      "dynamic.custom-approval.approval-flow-builtin.project-owner-workspace-dba-workspace-admin",
    roles: [
      PresetRoleType.PROJECT_OWNER,
      PresetRoleType.WORKSPACE_DBA,
      PresetRoleType.WORKSPACE_ADMIN,
    ],
  },
] as const;

// Generate title from roles (e.g., "Project Owner -> Workspace DBA")
const generateTitle = (roles: string[]): string => {
  return roles.map((role) => displayRoleTitle(role)).join(" -> ");
};

// Built-in approval flow templates
// These are materialized into the database only when actually used by rules
export const BUILTIN_APPROVAL_FLOWS: readonly BuiltinApprovalFlow[] =
  BUILTIN_FLOW_DEFINITIONS.map((def) => ({
    id: def.id,
    title: generateTitle(def.roles),
    description: t(def.descriptionKey),
    roles: def.roles,
    readonly: true as const,
  }));

export const BUILTIN_FLOW_ID_PREFIX = "bb.";

export const isBuiltinFlowId = (id: string): boolean => {
  return id.startsWith(BUILTIN_FLOW_ID_PREFIX);
};

export const getBuiltinFlow = (id: string): BuiltinApprovalFlow | undefined => {
  return BUILTIN_APPROVAL_FLOWS.find((flow) => flow.id === id);
};
