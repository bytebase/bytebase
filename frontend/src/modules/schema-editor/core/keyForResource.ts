import type {
  ColumnMetadata,
  Database,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";

export const keyForResource = (
  database: Database,
  metadata: {
    schema?: SchemaMetadata;
    table?: TableMetadata;
    column?: ColumnMetadata;
    partition?: TablePartitionMetadata;
    view?: ViewMetadata;
    procedure?: ProcedureMetadata;
    function?: FunctionMetadata;
  } = {},
  suffix?: string
) => {
  const {
    schema,
    table,
    view,
    procedure,
    function: func,
    column,
    partition,
  } = metadata;
  return keyForResourceName(
    {
      database: database.name,
      schema: schema?.name,
      table: table?.name,
      view: view?.name,
      procedure: procedure?.name,
      function: func?.name,
      column: column?.name,
      partition: partition?.name,
    },
    suffix
  );
};

export const keyForResourceName = (
  params: {
    database: string;
    schema?: string;
    table?: string;
    view?: string;
    procedure?: string;
    function?: string;
    column?: string;
    partition?: string;
  },
  suffix?: string
) => {
  const {
    database,
    schema,
    table,
    view,
    procedure,
    function: func,
    column,
    partition,
  } = params;
  const parts = [database];
  if (schema !== undefined) {
    parts.push(`schemas/${schema}`);
  }
  if (table !== undefined) {
    parts.push(`tables/${table}`);
  }
  if (view !== undefined) {
    parts.push(`views/${view}`);
  }
  if (procedure !== undefined) {
    parts.push(`procedures/${procedure}`);
  }
  if (func !== undefined) {
    parts.push(`functions/${func}`);
  }
  if (column !== undefined) {
    parts.push(`columns/${column}`);
  }
  if (partition !== undefined) {
    parts.push(`partitions/${partition}`);
  }
  if (suffix !== undefined) {
    parts.push(suffix);
  }

  return parts.join("/");
};
