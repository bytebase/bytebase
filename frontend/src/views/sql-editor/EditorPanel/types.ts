import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";

export type EditorPanelView =
  | "CODE"
  | "INFO"
  | "TABLES"
  | "VIEWS"
  | "FUNCTIONS"
  | "PROCEDURES"
  | "DIAGRAM";

export type BaseRichMetadata<T> = {
  type: T;
  database: DatabaseMetadata;
};
export type RichSchemaMetadata = BaseRichMetadata<"schema"> & {
  schema: SchemaMetadata;
};
export type RichTableMetadata = BaseRichMetadata<"table"> & {
  schema: SchemaMetadata;
  table: TableMetadata;
};
export type RichColumnMetadata = BaseRichMetadata<"column"> & {
  schema: SchemaMetadata;
  table: TableMetadata;
  column: ColumnMetadata;
};
export type RichViewMetadata = BaseRichMetadata<"view"> & {
  schema: SchemaMetadata;
  view: ViewMetadata;
};
export type RichFunctionMetadata = BaseRichMetadata<"function"> & {
  schema: SchemaMetadata;
  function: FunctionMetadata;
};
export type RichProcedureMetadata = BaseRichMetadata<"procedure"> & {
  schema: SchemaMetadata;
  procedure: ProcedureMetadata;
};

export type RichMetadataWithDB<T> = {
  db: ComposedDatabase;
  metadata: T extends "schema"
    ? RichSchemaMetadata
    : T extends "table"
      ? RichTableMetadata
      : T extends "column"
        ? RichColumnMetadata
        : T extends "view"
          ? RichViewMetadata
          : T extends "function"
            ? RichFunctionMetadata
            : T extends "procedure"
              ? RichProcedureMetadata
              : { type: T };
};

export type EditorPanelViewState = {
  view: EditorPanelView;
  schema?: string;
  pendingScroll?: RichMetadataWithDB<any>;
};

export const defaultViewState = (): EditorPanelViewState => {
  return {
    view: "CODE",
    schema: undefined,
  };
};

export const typeToView = (
  type: "table" | "view" | "function" | "procedure"
): EditorPanelView => {
  if (type === "table") return "TABLES";
  if (type === "view") return "VIEWS";
  if (type === "function") return "FUNCTIONS";
  if (type === "procedure") return "PROCEDURES";
  throw new Error(`unsupported type: "${type}"`);
};
