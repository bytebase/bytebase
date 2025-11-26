import { create } from "@bufbuild/protobuf";
import { v4 as uuidv4 } from "uuid";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { resolveCELExpr, wrapAsGroup } from "@/plugins/cel";
import type { LocalApprovalConfig, LocalApprovalRule } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type { ApprovalFlow } from "@/types/proto-es/v1/issue_service_pb";
import { ApprovalFlowSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { WorkspaceApprovalSetting } from "@/types/proto-es/v1/setting_service_pb";
import {
  WorkspaceApprovalSetting_Rule_Source,
  WorkspaceApprovalSetting_RuleSchema,
  WorkspaceApprovalSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { batchConvertCELStringToParsedExpr } from "@/utils";
import { displayRoleTitle } from "./role";

export const approvalNodeRoleText = (role: string) => {
  return displayRoleTitle(role);
};

// Convert proto WorkspaceApprovalSetting to local format
export const resolveLocalApprovalConfig = async (
  config: WorkspaceApprovalSetting
): Promise<LocalApprovalConfig> => {
  const rules: LocalApprovalRule[] = [];
  const expressions: string[] = [];
  const ruleIndices: number[] = [];

  for (let i = 0; i < config.rules.length; i++) {
    const protoRule = config.rules[i];
    const condition = protoRule.condition?.expression || "";

    const rule: LocalApprovalRule = {
      uid: uuidv4(),
      source:
        protoRule.source ||
        WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED,
      title: protoRule.template?.title || "",
      description: protoRule.template?.description || "",
      condition,
      flow:
        protoRule.template?.flow || create(ApprovalFlowSchema, { roles: [] }),
    };
    rules.push(rule);

    if (condition) {
      expressions.push(condition);
      ruleIndices.push(i);
    }
  }

  // Parse CEL expressions in batch
  if (expressions.length > 0) {
    const parsedExprs = await batchConvertCELStringToParsedExpr(expressions);
    for (let i = 0; i < parsedExprs.length; i++) {
      const ruleIndex = ruleIndices[i];
      const parsed = parsedExprs[i];
      if (parsed) {
        rules[ruleIndex].conditionExpr = wrapAsGroup(
          resolveCELExpr(parsed)
        ) as ConditionGroupExpr;
      }
    }
  }

  return { rules };
};

// Convert local format back to proto WorkspaceApprovalSetting
export const buildWorkspaceApprovalSetting = async (
  config: LocalApprovalConfig
): Promise<WorkspaceApprovalSetting> => {
  const protoRules = [];

  for (const rule of config.rules) {
    const protoRule = create(WorkspaceApprovalSetting_RuleSchema, {
      source: rule.source,
      condition: create(ExprSchema, { expression: rule.condition }),
      template: {
        flow: rule.flow,
        title: rule.title,
        description: rule.description,
      },
    });
    protoRules.push(protoRule);
  }

  return create(WorkspaceApprovalSettingSchema, {
    rules: protoRules,
  });
};

// Helper: Get rules filtered by source
export const getRulesBySource = (
  config: LocalApprovalConfig,
  source: WorkspaceApprovalSetting_Rule_Source
): LocalApprovalRule[] => {
  return config.rules.filter((r) => r.source === source);
};

// Helper: Format approval flow for display
export const formatApprovalFlow = (flow: ApprovalFlow): string => {
  if (!flow.roles || flow.roles.length === 0) {
    return "-";
  }
  return flow.roles.map((r) => displayRoleTitle(r)).join(" → ");
};
