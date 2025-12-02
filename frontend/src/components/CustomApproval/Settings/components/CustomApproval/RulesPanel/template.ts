import { create } from "@bufbuild/protobuf";
import { computed } from "vue";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { ExprType, wrapAsGroup } from "@/plugins/cel";
import { t } from "@/plugins/i18n";
import { PresetRoleType } from "@/types";
import { ApprovalFlowSchema } from "@/types/proto-es/v1/issue_service_pb";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import {
  CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS,
  CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
} from "@/utils/cel-attributes";

export type ApprovalRuleTemplate = {
  title: () => string;
  description: () => string;
  expr: ConditionGroupExpr;
  roles: string[];
  // If undefined, the template applies to all sources.
  // If specified, the template only applies to the listed sources.
  sources?: WorkspaceApprovalSetting_Rule_Source[];
};

export const useApprovalRuleTemplates = () => {
  const templates = computed((): ApprovalRuleTemplate[] => {
    return [
      {
        title: () =>
          t("custom-approval.approval-flow.template.presets.drop-or-truncate"),
        description: () =>
          t(
            "custom-approval.approval-flow.template.preset-descriptions.drop-or-truncate"
          ),
        // statement.sql_type in ["DROP_TABLE", "TRUNCATE"]
        expr: wrapAsGroup({
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_STATEMENT_SQL_TYPE, ["DROP_TABLE", "TRUNCATE"]],
        }),
        roles: [PresetRoleType.PROJECT_OWNER, PresetRoleType.WORKSPACE_DBA],
        sources: [WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE],
      },
      {
        title: () =>
          t(
            "custom-approval.approval-flow.template.presets.high-affected-rows"
          ),
        description: () =>
          t(
            "custom-approval.approval-flow.template.preset-descriptions.high-affected-rows"
          ),
        // statement.affected_rows > 100
        expr: wrapAsGroup({
          type: ExprType.Condition,
          operator: "_>_",
          args: [CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS, 100],
        }),
        roles: [PresetRoleType.PROJECT_OWNER, PresetRoleType.WORKSPACE_DBA],
        sources: [WorkspaceApprovalSetting_Rule_Source.CHANGE_DATABASE],
      },
      {
        title: () =>
          t("custom-approval.approval-flow.template.presets.fallback"),
        description: () =>
          t(
            "custom-approval.approval-flow.template.preset-descriptions.fallback"
          ),
        // The condition "true" matches all requests not matched by other rules.
        expr: wrapAsGroup({
          type: ExprType.RawString,
          content: "true",
        }),
        roles: [PresetRoleType.PROJECT_OWNER],
        // No sources specified - applies to all sources
      },
    ];
  });

  return templates;
};

export const filterTemplatesBySource = (
  templates: ApprovalRuleTemplate[],
  source: WorkspaceApprovalSetting_Rule_Source
): ApprovalRuleTemplate[] => {
  return templates.filter((template) => {
    // If no sources specified, the template applies to all sources
    if (!template.sources) {
      return true;
    }
    return template.sources.includes(source);
  });
};

export const applyTemplateToState = (
  template: ApprovalRuleTemplate
): {
  title: string;
  conditionExpr: ConditionGroupExpr;
  flow: ReturnType<typeof create<typeof ApprovalFlowSchema>>;
} => {
  return {
    title: template.title(),
    conditionExpr: template.expr,
    flow: create(ApprovalFlowSchema, { roles: [...template.roles] }),
  };
};
