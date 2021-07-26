import { Database } from "./database";
import { BackupId } from "./id";
import { Principal } from "./principal";

// Backup
export type Backup = {
  id: BackupId;

  // Related fields
  database: Database;

  // Standard fields
  creator: Principal;
  createdTs: number;

  name: string;
  path: string;
};
