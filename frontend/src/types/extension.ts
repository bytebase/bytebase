import { Database } from "./database";
import { ExtensionId } from "./id";
import { Principal } from "./principal";

// Extension
export type Extension = {
  id: ExtensionId;

  // Related fields
  database: Database;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  version: string;
  schema: string;
  description: string;
};
