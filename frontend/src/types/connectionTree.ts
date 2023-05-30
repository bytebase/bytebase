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
  parentId: string;
  id: string;
  key: string;
  label: string;
  type?: ConnectionAtomType;
  children?: ConnectionAtom[];
  disabled?: boolean;
  isLeaf?: boolean;
}
