import { create } from "@bufbuild/protobuf";
import { computed } from "vue";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { ExprType, wrapAsGroup } from "@/plugins/cel";
import { t } from "@/plugins/i18n";
import { PresetRoleType } from "@/types";
import { ApprovalFlowSchema } from "@/types/proto-es/v1/issue_service_pb";

export type ApprovalRuleTemplate = {
  title: () => string;
  description: () => string;
  expr: ConditionGroupExpr;
  roles: string[];
};

export const useApprovalRuleTemplates = () => {
  const templates = computed((): ApprovalRuleTemplate[] => {
    return [
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
      },
    ];
  });

  return templates;
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
