import { first, last } from "lodash-es";
import { computed } from "vue";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { ExprType, wrapAsGroup } from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";
import { useEnvironmentV1List } from "@/store";
import { PresetRiskLevel, PresetRoleType } from "@/types";
import { Risk_Source } from "@/types/proto/v1/risk_service";

/*
The risk for the production environment is considered to be high.
生产环境，默认为高风险。
environment == "prod"

The risk value for the development environment is considered to be low.
开发环境，默认为低风险
environment == "dev"

In the production environment, if the number of rows to be updated or deleted exceeds 10000, the risk is considered to be high.
生产环境中更新或删除的数据行数超过 100000，默认为高风险。
environment == "prod" & affected_rows > 10000 & sql_type in ["UPDATE", "DELETE"]

Creating a database in the production environment is considered to be a moderate risk.
在生产环境中创建数据库，默认为中风险。
environment == "prod"
*/

export type RuleTemplate = {
  key: string;
  expr: ConditionGroupExpr;
  level: number;
  source: Risk_Source;
};

export const useRuleTemplates = () => {
  const environmentList = useEnvironmentV1List();
  const dev = computed(() => first(environmentList.value));
  const prod = computed(() => last(environmentList.value));

  const ruleTemplateList = computed(() => {
    const templates: RuleTemplate[] = [
      {
        key: "request-role-high",
        expr: wrapAsGroup({
          type: ExprType.Condition,
          operator: "_==_",
          args: ["role", PresetRoleType.PROJECT_OWNER],
        }),
        level: PresetRiskLevel.HIGH,
        source: Risk_Source.REQUEST_ROLE,
      },
    ];
    if (prod.value) {
      templates.push(
        // environment == "prod" -> HIGH
        {
          key: "environment-prod-high",
          expr: wrapAsGroup({
            type: ExprType.Condition,
            operator: "_==_",
            args: ["environment_id", prod.value.id],
          }),
          level: PresetRiskLevel.HIGH,
          source: Risk_Source.SOURCE_UNSPECIFIED,
        },
        // environment == "prod" && affected_rows > 10000 && sql_type in ["UPDATE", "DELETE"]
        // -> HIGH
        {
          key: "dml-in-environment-prod-10k-rows-high",
          expr: wrapAsGroup({
            type: ExprType.ConditionGroup,
            operator: "_&&_",
            args: [
              {
                type: ExprType.Condition,
                operator: "_==_",
                args: ["environment_id", prod.value.id],
              },
              {
                type: ExprType.Condition,
                operator: "_>_",
                args: ["affected_rows", 10000],
              },
              {
                type: ExprType.Condition,
                operator: "@in",
                args: ["sql_type", ["UPDATE", "DELETE"]],
              },
            ],
          }),
          level: PresetRiskLevel.HIGH,
          source: Risk_Source.DML,
        },
        // create database
        // environment == "prod"
        // -> MODERATE
        {
          key: "create-database-in-environment-prod-moderate",
          expr: wrapAsGroup({
            type: ExprType.Condition,
            operator: "_==_",
            args: ["environment_id", prod.value.id],
          }),
          level: PresetRiskLevel.MODERATE,
          source: Risk_Source.CREATE_DATABASE,
        }
      );
    }

    if (dev.value && environmentList.value.length > 1) {
      // environment == "dev" -> LOW
      templates.push({
        key: "environment-dev-low",
        expr: wrapAsGroup({
          type: ExprType.Condition,
          operator: "_==_",
          args: ["environment_id", dev.value.id],
        }),
        level: PresetRiskLevel.LOW,
        source: Risk_Source.SOURCE_UNSPECIFIED,
      });
    }
    return templates;
  });

  return ruleTemplateList;
};

export const titleOfTemplate = (template: RuleTemplate) => {
  const { key } = template;
  const keypath = `dynamic.custom-approval.risk-rule.template.presets.${key}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return key;
};
