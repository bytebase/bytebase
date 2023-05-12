import { Schema } from "./schema";
import { SQLDialect } from "./sql";

export * from "./schema";
export * from "./sql";

export type ConnectionScope = "instance" | "database";

export type LanguageState = {
  schema: Schema;
  dialect: SQLDialect;
  connectionScope: ConnectionScope;
};
