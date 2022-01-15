import {
  DeploymentConfig,
  DeploymentSchedule,
  DeploymentSpec,
  Environment,
} from "../types";

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
