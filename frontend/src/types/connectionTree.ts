import { InstanceId, DatabaseId, TableId, ViewId } from "../types";

export type ConnectionAtomType = "instance" | "database" | "table" | "view";

export enum ConnectionTreeState {
  UNSET,
  LOADING,
  LOADED,
}

export interface ConnectionAtom {
  parentId: InstanceId | DatabaseId | TableId | ViewId;
  id: InstanceId | DatabaseId | TableId | ViewId;
  key: string;
  label: string;
  type?: ConnectionAtomType;
  children?: ConnectionAtom[];
  disabled?: boolean;
  isLeaf?: boolean;
}
