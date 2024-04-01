import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  ExternalTableMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";
import {
  useHoverStateContext as _useHoverStateContext,
  provideHoverStateContext as _provideHoverStateContext,
} from "../../../EditorCommon";

export const KEY = "schema-pane";

export type HoverState = {
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table?: TableMetadata;
  externalTable?: ExternalTableMetadata;
  view?: ViewMetadata;
  column?: ColumnMetadata;
};

export const useHoverStateContext = () => {
  return _useHoverStateContext<HoverState>(KEY);
};

export const provideHoverStateContext = () => {
  return _provideHoverStateContext<HoverState>(KEY);
};
