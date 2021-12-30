import { LabelId } from "./id";
import { Principal } from "./principal";

export type LabelKeyType = string;
export type LabelValueType = string;

export type Label = {
  id: LabelId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  key: LabelKeyType;
  valueList: LabelValueType[];
};

export type LabelPatch = {
  valueList?: LabelValueType[];
};
