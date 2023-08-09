import { first, last } from "lodash-es";
import { computed } from "vue";
import { ConditionGroupExpr, wrapAsGroup } from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";
import { useEnvironmentV1List } from "@/store";
import { PresetRiskLevel } from "@/types";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { extractEnvironmentResourceName } from "@/utils";

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
    const templates: RuleTemplate[] = [];
    if (prod.value) {
      // environment == "prod" -> HIGH
      templates.push({
        key: "environment-prod-high",
        expr: wrapAsGroup({
          operator: "_==_",
          args: [
            "environment_id",
            extractEnvironmentResourceName(prod.value.name),
          ],
        }),
        level: PresetRiskLevel.HIGH,
        source: Risk_Source.SOURCE_UNSPECIFIED,
      });
    }
    if (dev.value) {
      // environment == "dev" -> LOW
      templates.push({
        key: "environment-dev-low",
        expr: wrapAsGroup({
          operator: "_==_",
          args: [
            "environment_id",
            extractEnvironmentResourceName(dev.value.name),
          ],
        }),
        level: PresetRiskLevel.LOW,
        source: Risk_Source.SOURCE_UNSPECIFIED,
      });
    }
    if (prod.value) {
      // environment == "prod" && affected_rows > 10000 && sql_type in ["UPDATE", "DELETE"]
      // -> HIGH
      templates.push({
        key: "dml-in-environment-prod-10k-rows-high",
        expr: wrapAsGroup({
          operator: "_&&_",
          args: [
            {
              operator: "_==_",
              args: [
                "environment_id",
                extractEnvironmentResourceName(prod.value.name),
              ],
            },
            {
              operator: "_>_",
              args: ["affected_rows", 10000],
            },
            {
              operator: "@in",
              args: ["sql_type", ["UPDATE", "DELETE"]],
            },
          ],
        }),
        level: PresetRiskLevel.HIGH,
        source: Risk_Source.DML,
      });
    }
    if (prod.value) {
      // create database
      // environment == "prod"
      // -> MODERATE
      templates.push({
        key: "create-database-in-environment-prod-moderate",
        expr: wrapAsGroup({
          operator: "_==_",
          args: [
            "environment_id",
            extractEnvironmentResourceName(prod.value.name),
          ],
        }),
        level: PresetRiskLevel.MODERATE,
        source: Risk_Source.CREATE_DATABASE,
      });
    }
    return templates;
  });

  return ruleTemplateList;
};

export const titleOfTemplate = (template: RuleTemplate) => {
  const { key } = template;
  const keypath = `custom-approval.security-rule.template.presets.${key}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return key;
};
