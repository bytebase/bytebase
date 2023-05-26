import { ComposedDatabase } from "@/types";
import {
  LabelSelector,
  LabelSelectorRequirement,
  OperatorType,
  Schedule,
} from "@/types/proto/v1/project_service";

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

const checkLabelIn = (
  db: ComposedDatabase,
  rule: LabelSelectorRequirement
): boolean => {
  const value = db.labels[rule.key];
  if (!value) return false;

  return rule.values.includes(value);
};

const checkLabelExists = (
  db: ComposedDatabase,
  rule: LabelSelectorRequirement
): boolean => {
  return !!db.labels[rule.key];
};
