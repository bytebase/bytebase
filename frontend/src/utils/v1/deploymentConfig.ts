import { orderBy, uniq } from "lodash-es";
import { ComposedDatabase } from "@/types";
import {
  DeploymentConfig,
  DeploymentSpec,
  LabelSelector,
  LabelSelectorRequirement,
  OperatorType,
  Schedule,
} from "@/types/proto/v1/project_service";
import { getSemanticLabelValue } from "../label";

export const VIRTUAL_LABEL_KEYS = ["environment"];

export const extractDeploymentConfigName = (name: string) => {
  const pattern = /(?:^|\/)deploymentConfigs\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isVirtualLabelKey = (key: string) => {
  return VIRTUAL_LABEL_KEYS.includes(key);
};

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
  const envRule = rules.find((rule) => rule.key === "environment");
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

export const getAvailableDeploymentConfigMatchSelectorKeyList = (
  databaseList: ComposedDatabase[],
  withVirtualLabelKeys: boolean,
  sort = true
) => {
  const keys = uniq(databaseList.flatMap((db) => Object.keys(db.labels)));
  if (withVirtualLabelKeys) {
    VIRTUAL_LABEL_KEYS.forEach((key) => {
      if (!keys.includes(key)) keys.push(key);
    });
  }
  if (sort) {
    return orderBy(
      keys,
      [(key) => (isVirtualLabelKey(key) ? -1 : 1), (key) => key],
      ["asc", "asc"]
    );
  }
  return keys;
};

export const getPipelineFromDeploymentScheduleV1 = (
  databaseList: ComposedDatabase[],
  schedule: Schedule | undefined
): ComposedDatabase[][] => {
  const stages: ComposedDatabase[][] = [];

  const collectedIds = new Set<string>();
  schedule?.deployments.forEach((deployment) => {
    const dbs: ComposedDatabase[] = [];
    databaseList.forEach((db) => {
      if (collectedIds.has(db.uid)) return;
      if (isDatabaseMatchesSelectorV1(db, deployment.spec?.labelSelector)) {
        dbs.push(db);
        collectedIds.add(db.uid);
      }
    });
    stages.push(dbs);
  });

  return stages;
};

export const isDatabaseMatchesSelectorV1 = (
  database: ComposedDatabase,
  selector: LabelSelector | undefined
): boolean => {
  const rules = selector?.matchExpressions ?? [];
  return rules.every((rule) => {
    switch (rule.operator) {
      case OperatorType.OPERATOR_TYPE_IN:
        return checkLabelIn(database, rule);
      case OperatorType.OPERATOR_TYPE_EXISTS:
        return checkLabelExists(database, rule);
      default:
        // unknown operators are taken as mismatch
        console.warn(`known operator "${rule.operator}"`);
        return false;
    }
  });
};

export const displayDeploymentMatchSelectorKey = (key: string) => {
  if (key === "environment") {
    return "Environment ID";
  }

  return key;
};

const checkLabelIn = (
  db: ComposedDatabase,
  rule: LabelSelectorRequirement
): boolean => {
  const value = getSemanticLabelValue(db, rule.key);
  if (!value) return false;

  return rule.values.includes(value);
};

const checkLabelExists = (
  db: ComposedDatabase,
  rule: LabelSelectorRequirement
): boolean => {
  return !!getSemanticLabelValue(db, rule.key);
};
