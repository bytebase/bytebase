import {
  DeploymentConfig,
  DeploymentSchedule,
  DeploymentSpec,
  Environment,
  Label,
} from "../types";
import escapeStringRegexp from "escape-string-regexp";
import { hidePrefix } from "./label";

export const generateDefaultSchedule = (environmentList: Environment[]) => {
  const schedule: DeploymentSchedule = {
    deployments: [],
  };
  environmentList.forEach((env) => {
    schedule.deployments.push({
      name: `${env.name} Stage`,
      spec: {
        selector: {
          matchExpressions: [
            {
              key: "bb.environment",
              operator: "In",
              values: [env.name],
            },
          ],
        },
      },
    });
  });
  return schedule;
};

export const validateDeploymentConfig = (
  config: DeploymentConfig
): string | undefined => {
  const { deployments } = config.schedule;
  if (deployments.length === 0) {
    return "deployment-config.error.at-least-one-stage";
  }

  for (let i = 0; i < config.schedule.deployments.length; i++) {
    const deployment = config.schedule.deployments[i];
    if (!deployment.name.trim()) {
      return "deployment-config.error.stage-name-required";
    }
    const error = validateDeploymentSpec(deployment.spec);
    if (error) return error;
  }

  return undefined;
};

export const validateDeploymentSpec = (
  deployment: DeploymentSpec
): string | undefined => {
  const rules = deployment.selector.matchExpressions;
  if (rules.length === 0) {
    return "deployment-config.error.at-least-one-selector";
  }
  const envRule = rules.find((rule) => rule.key === "bb.environment");
  if (!envRule || envRule.operator !== "In") {
    return "deployment-config.error.env-in-selector-required";
  }
  if (envRule.values.length !== 1) {
    return "deployment-config.error.env-selector-must-has-one-value";
  }

  for (let i = 0; i < rules.length; i++) {
    const rule = rules[i];
    if (!rule.key) {
      return "deployment-config.error.key-required";
    }
    if (rule.operator === "In" && rule.values.length === 0) {
      return "deployment-config.error.values-required";
    }
  }
  return undefined;
};

export const parseDatabaseNameByTemplate = (
  name: string,
  template: string,
  labelList: Label[]
) => {
  let regexpString = template.replace("{{DB_NAME}}", "(?<name>.+?)");
  /*
    Rewrite the placeholder-based template to a big RegExp
    e.g. template = "{{DB_NAME}}_{{TENANT}}"
    bb.tenant has values (bytebase, tenant1, tenant2)
    here regex will be /^(?<name>.+?)_(bytebase|tenant1|tenant2)$/
  */
  labelList.forEach((label) => {
    const { key, valueList } = label;
    const placeholder = `{{${hidePrefix(key).toUpperCase()}}}`;
    // replace special chars in values
    const escapedValueList = valueList.map((value) =>
      escapeStringRegexp(value)
    );
    const regex = `(${escapedValueList.join("|")})`;
    regexpString = regexpString.replace(placeholder, regex);
  });
  const regex = new RegExp(`^${regexpString}$`);
  const match = name.match(regex);

  // fallback to name it self when failed
  return match?.groups?.name || name;
};
