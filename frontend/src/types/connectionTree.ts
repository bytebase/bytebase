import { InstanceId, DatabaseId, ProjectId } from "../types";

export type ConnectionAtomType = "project" | "instance" | "database";

export enum ConnectionTreeState {
  UNSET,
  LOADING,
  LOADED,
}

export enum ConnectionTreeMode {
  PROJECT = "project",
  INSTANCE = "instance",
}

export interface ConnectionAtom {
  parentId: ProjectId | InstanceId | DatabaseId;
  id: ProjectId | InstanceId | DatabaseId;
  key: string;
  label: string;
  type?: ConnectionAtomType;
  children?: ConnectionAtom[];
  disabled?: boolean;
  isLeaf?: boolean;
}
