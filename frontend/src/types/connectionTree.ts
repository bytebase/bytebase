import { InstanceId, DatabaseId } from "../types";

export type ConnectionAtomType = "instance" | "database" | "table" | "view";

export enum ConnectionTreeState {
  UNSET,
  LOADING,
  LOADED,
}

export interface ConnectionAtom {
  parentId: InstanceId | DatabaseId;
  id: InstanceId | DatabaseId;
  key: string;
  label: string;
  type?: ConnectionAtomType;
  children?: ConnectionAtom[];
  disabled?: boolean;
  isLeaf?: boolean;
}
