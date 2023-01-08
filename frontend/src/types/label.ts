import { LabelId } from "./id";

export type LabelKeyType = string;
export type LabelValueType = string;

export type Label = {
  id: LabelId;

  // Domain specific fields
  key: LabelKeyType;
  valueList: LabelValueType[];
};

export type LabelPatch = {
  valueList?: LabelValueType[];
};

export type DatabaseLabel = {
  key: LabelKeyType;
  value: LabelValueType;
};

export type AvailableLabel = {
  key: LabelKeyType;
  valueList: LabelValueType[];
};
