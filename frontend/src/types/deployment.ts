import { DeploymentConfigId } from "./id";
import { LabelKeyType, LabelValueType } from "./label";

export type DeploymentConfig = {
  id: DeploymentConfigId;

  // Domain specific fields
  schedule: DeploymentSchedule;
};

export type DeploymentConfigPatch = {
  payload: string;
};

export type DeploymentSchedule = {
  deployments: Deployment[];
};

export type Deployment = {
  name: string;
  spec: DeploymentSpec;
};

export type DeploymentSpec = {
  selector: LabelSelector;
};

export type LabelSelector = {
  matchExpressions: LabelSelectorRequirement[];
};

export type LabelSelectorRequirement = {
  key: LabelKeyType;
  operator: OperatorType;
  values: LabelValueType[];
};

export type OperatorType = "In" | "Exists";
