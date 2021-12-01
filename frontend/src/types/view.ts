import { Database } from "./database";
import { ViewId } from "./id";
import { Principal } from "./principal";

// View
export type View = {
  id: ViewId;

  // Related fields
  database: Database;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  definition: string;
  comment: string;
};
