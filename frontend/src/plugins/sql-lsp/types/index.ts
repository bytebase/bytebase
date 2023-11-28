import type { SQLDialect } from "@/types";
import type { Schema } from "./schema";

export * from "./monaco-editor";
export * from "./schema";

export type ConnectionScope = "instance" | "database";

export type LanguageState = {
  schema: Schema;
  dialect: SQLDialect;
  connectionScope: ConnectionScope;
};
