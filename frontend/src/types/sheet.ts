import {
  SheetId,
  Instance,
  InstanceId,
  Database,
  DatabaseId,
  Principal,
} from ".";

export type SheetVisibility = "PRIVATE" | "PROJECT" | "PUBLIC";

export interface Sheet {
  id: SheetId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Related fields
  instanceId: InstanceId;
  instance: Instance;
  databaseId?: DatabaseId;
  database?: Database;

  // Domain fields
  name: string;
  statement: string;
  visibility: SheetVisibility;
}

export type CreateSheetState = Omit<
  Sheet,
  "id" | "creator" | "createdTs" | "updater" | "updatedTs"
>;

export type SheetPatch = Partial<
  Pick<Sheet, "id" | "name" | "statement" | "visibility">
>;

export type AccessOption = {
  label: string;
  description: string;
  value: SheetVisibility;
};
