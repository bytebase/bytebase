import { countBy, groupBy, uniqBy } from "lodash-es";
import {
  Database,
  DatabaseLabel,
  DeploymentSchedule,
  Label,
  LabelKeyType,
  LabelSelector,
  LabelSelectorRequirement,
  LabelValueType,
} from "../types";

// reserved labels (e.g. bb.environment) have zero ID and their values are immutable.
// see api/label.go for more details
export const RESERVED_LABEL_ID = 0;

export const isReservedLabel = (label: Label): boolean => {
  return label.id === RESERVED_LABEL_ID;
};

export const isReservedDatabaseLabel = (
  dbLabel: DatabaseLabel,
  labelList: Label[]
): boolean => {
  const label = labelList.find((label) => label.key === dbLabel.key);
  if (!label) return false;
  return label.id === RESERVED_LABEL_ID;
};

export const getLabelValue = (
  db: Database,
  key: LabelKeyType
): LabelValueType => {
  const label = db.labels.find((target) => target.key === key);
  if (!label) return "";
  return label.value;
};

export const groupingDatabaseListByLabelKey = (
  databaseList: Database[],
  key: LabelKeyType,
  emptyValue: LabelValueType = ""
): Array<{ labelValue: LabelValueType; databaseList: Database[] }> => {
  const dict = groupBy(databaseList, (db) => {
    const label = db.labels.find((target) => target.key === key);
    if (!label) return emptyValue;
    return label.value;
  });
  return Object.keys(dict).map((value) => ({
    labelValue: value,
    databaseList: dict[value],
  }));
};

export const validateLabels = (labels: DatabaseLabel[]): string | undefined => {
  for (let i = 0; i < labels.length; i++) {
    const label = labels[i];
    if (!label.key) return "label.error.key-necessary";
    if (!label.value) return "label.error.value-necessary";
  }
  if (labels.length !== uniqBy(labels, "key").length) {
    return "label.error.key-duplicated";
  }
  return undefined;
};

export const findDefaultGroupByLabel = (
  labelList: Label[],
  databaseList: Database[]
): string | undefined => {
  const availableKeys = labelList.map((label) => label.key);

  // concat all databases' keys into one array
  const databaseLabelKeys = databaseList.flatMap((db) =>
    db.labels
      .map((label) => label.key)
      .filter((key) => availableKeys.includes(key))
  );
  if (databaseLabelKeys.length > 0) {
    // counting up the keys' frequency
    const countsDict = countBy(databaseLabelKeys);
    const countsList = Object.keys(countsDict).map((key) => ({
      key,
      count: countsDict[key],
    }));
    // return the most frequent used key
    countsList.sort((a, b) => b.count - a.count);
    return countsList[0].key;
  } else {
    // just use the first label key
    return availableKeys[0];
  }
};

export const filterDatabaseListByLabelSelector = (
  databaseList: Database[],
  labelSelector: LabelSelector
): Database[] => {
  return databaseList.filter((db) =>
    isDatabaseMatchesSelector(db, labelSelector)
  );
};

export const getPipelineFromDeploymentSchedule = (
  databaseList: Database[],
  schedule: DeploymentSchedule
): Database[][] => {
  const stages: Database[][] = [];

  const collectedIds = new Set<Database["id"]>();
  schedule.deployments.forEach((deployment) => {
    const dbs: Database[] = [];
    databaseList.forEach((db) => {
      if (collectedIds.has(db.id)) return;
      if (isDatabaseMatchesSelector(db, deployment.spec.selector)) {
        dbs.push(db);
        collectedIds.add(db.id);
      }
    });
    stages.push(dbs);
  });

  return stages;
};

export const isDatabaseMatchesSelector = (
  database: Database,
  selector: LabelSelector
): boolean => {
  const rules = selector.matchExpressions;
  return rules.every((rule) => {
    switch (rule.operator) {
      case "In":
        return checkLabelIn(database, rule);
      case "Exists":
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
