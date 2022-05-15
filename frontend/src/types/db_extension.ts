import { Database } from "./database";
import { DBExtensionId } from "./id";
import { Principal } from "./principal";

// DBExtension
export type DBExtension = {
  id: DBExtensionId;

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
