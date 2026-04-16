export type {
  CommonTabContext,
  CoreTabContext,
  EditStatus,
  EditTarget,
  RolloutObject,
  TabContext,
  TabType,
} from "@/components/SchemaEditorLite/types";

import type { CoreTabContext } from "@/components/SchemaEditorLite/types";

export type { RebuildMetadataEditReset } from "@/components/SchemaEditorLite/algorithm/rebuild";

import type { RebuildMetadataEditReset } from "@/components/SchemaEditorLite/algorithm/rebuild";
import type {
  EditStatus,
  EditTarget,
  RolloutObject,
  TabContext,
} from "@/components/SchemaEditorLite/types";
import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

export type SchemaEditorOptions = {
  forceShowIndexes: boolean;
  forceShowPartitions: boolean;
};

export type ResourceMetadata = {
  schema?: SchemaMetadata;
  table?: TableMetadata;
  column?: ColumnMetadata;
  partition?: TablePartitionMetadata;
  view?: ViewMetadata;
  procedure?: ProcedureMetadata;
  function?: FunctionMetadata;
};

export type SchemaResourceMetadata = {
  schema: SchemaMetadata;
  table?: TableMetadata;
  column?: ColumnMetadata;
  partition?: TablePartitionMetadata;
  procedure?: ProcedureMetadata;
  function?: FunctionMetadata;
  view?: ViewMetadata;
};

// Scroll status rich metadata types
export type RichSchemaMetadata = {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
};

export type RichTableMetadata = RichSchemaMetadata & {
  table: TableMetadata;
};

export type RichColumnMetadata = RichTableMetadata & {
  column: ColumnMetadata;
};

export type RichMetadataWithDB<T> = {
  db: Database;
  metadata: T;
};

// Selection state
export type SelectionState = {
  checked: boolean;
  indeterminate: boolean;
};

// Tabs context
export type TabsContext = {
  tabList: TabContext[];
  currentTabId: string;
  currentTab: TabContext | undefined;
  addTab: (coreTab: CoreTabContext, setAsCurrentTab?: boolean) => void;
  setCurrentTab: (id: string) => void;
  closeTab: (id: string) => void;
  findTab: (target: CoreTabContext) => TabContext | undefined;
  clearTabs: () => void;
};

// Edit status context
export type EditStatusContext = {
  isDirty: boolean;
  markEditStatus: (
    database: Database,
    metadata: SchemaResourceMetadata,
    status: EditStatus
  ) => void;
  markEditStatusByKey: (key: string, status: EditStatus) => void;
  getEditStatusByKey: (key: string) => EditStatus | undefined;
  removeEditStatus: (
    database: Database,
    metadata: SchemaResourceMetadata,
    recursive: boolean
  ) => void;
  clearEditStatus: () => void;
  getSchemaStatus: (
    database: Database,
    metadata: { schema: SchemaMetadata }
  ) => EditStatus;
  getTableStatus: (
    database: Database,
    metadata: { schema: SchemaMetadata; table: TableMetadata }
  ) => EditStatus;
  getColumnStatus: (
    database: Database,
    metadata: {
      schema: SchemaMetadata;
      table: TableMetadata;
      column: ColumnMetadata;
    }
  ) => EditStatus;
  getPartitionStatus: (
    database: Database,
    metadata: {
      schema: SchemaMetadata;
      table: TableMetadata;
      partition: TablePartitionMetadata;
    }
  ) => EditStatus;
  getProcedureStatus: (
    database: Database,
    metadata: { schema: SchemaMetadata; procedure: ProcedureMetadata }
  ) => EditStatus;
  getFunctionStatus: (
    database: Database,
    metadata: { schema: SchemaMetadata; function: FunctionMetadata }
  ) => EditStatus;
  getViewStatus: (
    database: Database,
    metadata: { schema: SchemaMetadata; view: ViewMetadata }
  ) => EditStatus;
  replaceTableName: (
    database: Database,
    metadata: { schema: SchemaMetadata; table: TableMetadata },
    newName: string
  ) => void;
};

// Selection context
export type SelectionContext = {
  selectionEnabled: boolean;
  getTableSelectionState: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    }
  ) => SelectionState;
  updateTableSelection: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    on: boolean
  ) => void;
  getAllTablesSelectionState: (
    db: Database,
    metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
    tables: TableMetadata[]
  ) => SelectionState;
  updateAllTablesSelection: (
    db: Database,
    metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
    tables: TableMetadata[],
    on: boolean
  ) => void;
  getColumnSelectionState: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      column: ColumnMetadata;
    }
  ) => SelectionState;
  updateColumnSelection: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      column: ColumnMetadata;
    },
    on: boolean
  ) => void;
  getAllColumnsSelectionState: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    columns: ColumnMetadata[]
  ) => SelectionState;
  updateAllColumnsSelection: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    columns: ColumnMetadata[],
    on: boolean
  ) => void;
  getViewSelectionState: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      view: ViewMetadata;
    }
  ) => SelectionState;
  updateViewSelection: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      view: ViewMetadata;
    },
    on: boolean
  ) => void;
  getAllViewsSelectionState: (
    db: Database,
    metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
    views: ViewMetadata[]
  ) => SelectionState;
  updateAllViewsSelection: (
    db: Database,
    metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
    views: ViewMetadata[],
    on: boolean
  ) => void;
  getProcedureSelectionState: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      procedure: ProcedureMetadata;
    }
  ) => SelectionState;
  updateProcedureSelection: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      procedure: ProcedureMetadata;
    },
    on: boolean
  ) => void;
  getAllProceduresSelectionState: (
    db: Database,
    metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
    procedures: ProcedureMetadata[]
  ) => SelectionState;
  updateAllProceduresSelection: (
    db: Database,
    metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
    procedures: ProcedureMetadata[],
    on: boolean
  ) => void;
  getFunctionSelectionState: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      function: FunctionMetadata;
    }
  ) => SelectionState;
  updateFunctionSelection: (
    db: Database,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      function: FunctionMetadata;
    },
    on: boolean
  ) => void;
  getAllFunctionsSelectionState: (
    db: Database,
    metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
    functions: FunctionMetadata[]
  ) => SelectionState;
  updateAllFunctionsSelection: (
    db: Database,
    metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
    functions: FunctionMetadata[],
    on: boolean
  ) => void;
};

// Scroll status context
export type ScrollStatusContext = {
  pendingScrollToTable: RichMetadataWithDB<RichTableMetadata> | undefined;
  pendingScrollToColumn: RichMetadataWithDB<RichColumnMetadata> | undefined;
  queuePendingScrollToTable: (
    params: RichMetadataWithDB<RichTableMetadata>
  ) => void;
  queuePendingScrollToColumn: (
    params: RichMetadataWithDB<RichColumnMetadata>
  ) => void;
  consumePendingScrollToTable: () => void;
  consumePendingScrollToColumn: () => void;
};

// Combined context value
export type SchemaEditorContextValue = {
  // Props
  readonly: boolean;
  project: Project;
  targets: EditTarget[];
  hidePreview: boolean;
  options?: SchemaEditorOptions;

  // Sub-contexts
  tabs: TabsContext;
  editStatus: EditStatusContext;
  selection: SelectionContext;
  scrollStatus: ScrollStatusContext;

  // Callbacks (replacing Emittery events)
  rebuildTree: (openFirstChild: boolean) => void;
  rebuildEditStatus: (resets: RebuildMetadataEditReset[]) => void;
  refreshPreview: () => void;
  mergeMetadata: (metadatas: DatabaseMetadata[]) => void;

  // Algorithm
  applyMetadataEdit: (
    database: Database,
    metadata: DatabaseMetadata
  ) => { metadata: DatabaseMetadata };
  rebuildMetadataEdit: (
    target: EditTarget,
    resets?: RebuildMetadataEditReset[]
  ) => void;
};

// Props for the top-level component
export type SchemaEditorProps = {
  project: Project;
  readonly?: boolean;
  selectedRolloutObjects?: RolloutObject[];
  targets?: EditTarget[];
  loading?: boolean;
  hidePreview?: boolean;
  options?: SchemaEditorOptions;
  onSelectedRolloutObjectsChange?: (objects: RolloutObject[]) => void;
  onIsEditingChange?: (objects: RolloutObject[]) => void;
};

// Imperative handle exposed via ref
export type SchemaEditorHandle = {
  applyMetadataEdit: (
    database: Database,
    metadata: DatabaseMetadata
  ) => { metadata: DatabaseMetadata };
  refreshPreview: () => void;
  isDirty: boolean;
};
