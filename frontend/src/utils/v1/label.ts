import { Database } from "@/types";
import {
  LabelSelector,
  LabelSelectorRequirement,
  OperatorType,
  Schedule,
} from "@/types/proto/v1/project_service";

export const getPipelineFromDeploymentScheduleV1 = (
  databaseList: Database[],
  schedule: Schedule | undefined
): Database[][] => {
  const stages: Database[][] = [];

  const collectedIds = new Set<Database["id"]>();
  schedule?.deployments.forEach((deployment) => {
    const dbs: Database[] = [];
    databaseList.forEach((db) => {
      if (collectedIds.has(db.id)) return;
      if (isDatabaseMatchesSelectorV1(db, deployment.spec?.labelSelector)) {
        dbs.push(db);
        collectedIds.add(db.id);
      }
    });
    stages.push(dbs);
  });

  return stages;
};

export const isDatabaseMatchesSelectorV1 = (
  database: Database,
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
  db: Database,
  rule: LabelSelectorRequirement
): boolean => {
  const label = db.labels.find((label) => label.key === rule.key);
  if (!label) return false;

  return rule.values.some((value) => value === label.value);
};

const checkLabelExists = (
  db: Database,
  rule: LabelSelectorRequirement
): boolean => {
  return db.labels.some((label) => label.key === rule.key);
};
