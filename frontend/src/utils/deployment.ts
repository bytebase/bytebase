import type { DeploymentConfig, DeploymentSpec } from "@/types";
import { PRESET_DB_NAME_TEMPLATE_PLACEHOLDERS } from "./label";

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

export const buildDatabaseNameRegExpByTemplate = (template: string): RegExp => {
  let regexpString = template;

  /*
    Rewrite the placeholder-based template to a big RegExp
    e.g. template = "{{DB_NAME}}__{{TENANT}}"
    here regex will be /^(?<DB_NAME>.+?)__(?<TENANT>.+?)$/
  */
  PRESET_DB_NAME_TEMPLATE_PLACEHOLDERS.forEach((placeholder) => {
    const pattern = `{{${placeholder}}}`;
    const groupRegExp = `(?<${placeholder}>.+?)`;
    regexpString = regexpString.replace(pattern, groupRegExp);
  });

  const regexp = new RegExp(`^${regexpString}$`);
  return regexp;
};

export const parseDatabaseLabelValueByTemplate = (
  template: string,
  name: string,
  group: "DB_NAME" | "TENANT"
) => {
  if (!template) return "";

  const regex = buildDatabaseNameRegExpByTemplate(template);
  const matches = name.match(regex);
  if (!matches) return "";

  const value = matches.groups?.[group];
  if (!value) return "";
  return value;
};
