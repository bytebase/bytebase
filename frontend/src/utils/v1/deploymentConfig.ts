import {
  DeploymentConfig,
  DeploymentSpec,
  OperatorType,
} from "@/types/proto/v1/project_service";

export const validateDeploymentConfigV1 = (
  config: DeploymentConfig
): string | undefined => {
  const deployments = config.schedule?.deployments ?? [];
  if (deployments.length === 0) {
    return "deployment-config.error.at-least-one-stage";
  }

  for (let i = 0; i < deployments.length; i++) {
    const deployment = deployments[i];
    if (!deployment.title.trim()) {
      return "deployment-config.error.stage-name-required";
    }
    const error = validateDeploymentSpecV1(deployment.spec);
    if (error) return error;
  }

  return undefined;
};

export const validateDeploymentSpecV1 = (
  spec: DeploymentSpec | undefined
): string | undefined => {
  const rules = spec?.labelSelector?.matchExpressions ?? [];
  if (rules.length === 0) {
    return "deployment-config.error.at-least-one-selector";
  }
  const envRule = rules.find((rule) => rule.key === "bb.environment");
  if (!envRule || envRule.operator !== OperatorType.OPERATOR_TYPE_IN) {
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
    if (
      rule.operator === OperatorType.OPERATOR_TYPE_IN &&
      rule.values.length === 0
    ) {
      return "deployment-config.error.values-required";
    }
  }
  return undefined;
};
