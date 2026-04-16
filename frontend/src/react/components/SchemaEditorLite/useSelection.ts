import { pick } from "lodash-es";
import { useCallback, useMemo } from "react";
import { keyForResource } from "@/components/SchemaEditorLite/context/common";
import type { RolloutObject } from "@/components/SchemaEditorLite/types";
import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { SelectionContext, SelectionState } from "./types";

const EMPTY_STATE: SelectionState = { checked: false, indeterminate: false };

export function useSelection(
  selectedRolloutObjects: RolloutObject[] | undefined,
  onSelectedRolloutObjectsChange?: (objects: RolloutObject[]) => void
): SelectionContext {
  const selectionEnabled = selectedRolloutObjects !== undefined;

  const selectedRolloutObjectMap = useMemo(() => {
    return new Map(
      selectedRolloutObjects?.map((ro) => {
        const key = keyForResource(ro.db, ro.metadata);
        return [key, ro] as const;
      })
    );
  }, [selectedRolloutObjects]);

  const emit = useCallback(
    (map: Map<string, RolloutObject>) => {
      onSelectedRolloutObjectsChange?.(Array.from(map.values()));
    },
    [onSelectedRolloutObjectsChange]
  );

  // --- Table ---

  const updateTableSelectionImpl = useCallback(
    (
      map: Map<string, RolloutObject>,
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        table: TableMetadata;
      },
      on: boolean
    ) => {
      const { table } = metadata;
      if (on) {
        map.set(keyForResource(db, metadata), { db, metadata });
        table.columns.forEach((column) => {
          map.set(keyForResource(db, { ...metadata, column }), {
            db,
            metadata: { ...metadata, column },
          });
        });
      } else {
        map.delete(keyForResource(db, metadata));
        table.columns.forEach((column) => {
          map.delete(keyForResource(db, { ...metadata, column }));
        });
      }
    },
    []
  );

  const updateColumnSelectionImpl = useCallback(
    (
      map: Map<string, RolloutObject>,
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        table: TableMetadata;
        column: ColumnMetadata;
      },
      on: boolean
    ) => {
      const tableMeta = pick(metadata, "database", "schema", "table");
      if (on) {
        map.set(keyForResource(db, tableMeta), { db, metadata: tableMeta });
        map.set(keyForResource(db, metadata), { db, metadata });
      } else {
        map.delete(keyForResource(db, metadata));
      }
    },
    []
  );

  const updateSimpleSelectionImpl = useCallback(
    (
      map: Map<string, RolloutObject>,
      db: Database,
      metadata: Record<string, unknown>,
      on: boolean
    ) => {
      const key = keyForResource(
        db,
        metadata as Parameters<typeof keyForResource>[1]
      );
      if (on) {
        map.set(key, {
          db,
          metadata: metadata as RolloutObject["metadata"],
        });
      } else {
        map.delete(key);
      }
    },
    []
  );

  // Table selection
  const getTableSelectionState = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        table: TableMetadata;
      }
    ): SelectionState => {
      if (!selectionEnabled) return EMPTY_STATE;
      const { table } = metadata;
      const tableChecked = selectedRolloutObjectMap.has(
        keyForResource(db, metadata)
      );
      if (table.columns.length === 0) {
        return { checked: tableChecked, indeterminate: false };
      }
      const checkedColumns = table.columns.filter((column) =>
        selectedRolloutObjectMap.has(
          keyForResource(db, { ...metadata, column })
        )
      );
      const checked = checkedColumns.length === table.columns.length;
      return {
        checked,
        indeterminate: !checked && (tableChecked || checkedColumns.length > 0),
      };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateTableSelection = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        table: TableMetadata;
      },
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      updateTableSelectionImpl(updatedMap, db, metadata, on);
      emit(updatedMap);
    },
    [selectionEnabled, selectedRolloutObjectMap, updateTableSelectionImpl, emit]
  );

  const getAllTablesSelectionState = useCallback(
    (
      db: Database,
      metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
      tables: TableMetadata[]
    ): SelectionState => {
      if (!selectionEnabled || tables.length === 0) return EMPTY_STATE;
      const selected = tables.filter((table) =>
        selectedRolloutObjectMap.has(keyForResource(db, { ...metadata, table }))
      );
      return {
        checked: selected.length === tables.length,
        indeterminate: selected.length > 0 && selected.length < tables.length,
      };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateAllTablesSelection = useCallback(
    (
      db: Database,
      metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
      tables: TableMetadata[],
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      tables.forEach((table) => {
        updateTableSelectionImpl(updatedMap, db, { ...metadata, table }, on);
      });
      emit(updatedMap);
    },
    [selectionEnabled, selectedRolloutObjectMap, updateTableSelectionImpl, emit]
  );

  // Column selection
  const getColumnSelectionState = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        table: TableMetadata;
        column: ColumnMetadata;
      }
    ): SelectionState => {
      if (!selectionEnabled) return EMPTY_STATE;
      const checked = selectedRolloutObjectMap.has(
        keyForResource(db, metadata)
      );
      return { checked, indeterminate: false };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateColumnSelection = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        table: TableMetadata;
        column: ColumnMetadata;
      },
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      updateColumnSelectionImpl(updatedMap, db, metadata, on);
      emit(updatedMap);
    },
    [
      selectionEnabled,
      selectedRolloutObjectMap,
      updateColumnSelectionImpl,
      emit,
    ]
  );

  const getAllColumnsSelectionState = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        table: TableMetadata;
      },
      columns: ColumnMetadata[]
    ): SelectionState => {
      if (!selectionEnabled || columns.length === 0) return EMPTY_STATE;
      const selected = columns.filter((column) =>
        selectedRolloutObjectMap.has(
          keyForResource(db, { ...metadata, column })
        )
      );
      return {
        checked: selected.length === columns.length,
        indeterminate: selected.length > 0 && selected.length < columns.length,
      };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateAllColumnsSelection = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        table: TableMetadata;
      },
      columns: ColumnMetadata[],
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      columns.forEach((column) => {
        updateColumnSelectionImpl(updatedMap, db, { ...metadata, column }, on);
      });
      emit(updatedMap);
    },
    [
      selectionEnabled,
      selectedRolloutObjectMap,
      updateColumnSelectionImpl,
      emit,
    ]
  );

  // View selection
  const getViewSelectionState = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        view: ViewMetadata;
      }
    ): SelectionState => {
      if (!selectionEnabled) return EMPTY_STATE;
      return {
        checked: selectedRolloutObjectMap.has(keyForResource(db, metadata)),
        indeterminate: false,
      };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateViewSelection = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        view: ViewMetadata;
      },
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      updateSimpleSelectionImpl(updatedMap, db, metadata, on);
      emit(updatedMap);
    },
    [
      selectionEnabled,
      selectedRolloutObjectMap,
      updateSimpleSelectionImpl,
      emit,
    ]
  );

  const getAllViewsSelectionState = useCallback(
    (
      db: Database,
      metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
      views: ViewMetadata[]
    ): SelectionState => {
      if (!selectionEnabled || views.length === 0) return EMPTY_STATE;
      const selected = views.filter((view) =>
        selectedRolloutObjectMap.has(keyForResource(db, { ...metadata, view }))
      );
      return {
        checked: selected.length === views.length,
        indeterminate: selected.length > 0 && selected.length < views.length,
      };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateAllViewsSelection = useCallback(
    (
      db: Database,
      metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
      views: ViewMetadata[],
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      views.forEach((view) => {
        updateSimpleSelectionImpl(updatedMap, db, { ...metadata, view }, on);
      });
      emit(updatedMap);
    },
    [
      selectionEnabled,
      selectedRolloutObjectMap,
      updateSimpleSelectionImpl,
      emit,
    ]
  );

  // Procedure selection
  const getProcedureSelectionState = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        procedure: ProcedureMetadata;
      }
    ): SelectionState => {
      if (!selectionEnabled) return EMPTY_STATE;
      return {
        checked: selectedRolloutObjectMap.has(keyForResource(db, metadata)),
        indeterminate: false,
      };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateProcedureSelection = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        procedure: ProcedureMetadata;
      },
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      updateSimpleSelectionImpl(updatedMap, db, metadata, on);
      emit(updatedMap);
    },
    [
      selectionEnabled,
      selectedRolloutObjectMap,
      updateSimpleSelectionImpl,
      emit,
    ]
  );

  const getAllProceduresSelectionState = useCallback(
    (
      db: Database,
      metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
      procedures: ProcedureMetadata[]
    ): SelectionState => {
      if (!selectionEnabled || procedures.length === 0) return EMPTY_STATE;
      const selected = procedures.filter((procedure) =>
        selectedRolloutObjectMap.has(
          keyForResource(db, { ...metadata, procedure })
        )
      );
      return {
        checked: selected.length === procedures.length,
        indeterminate:
          selected.length > 0 && selected.length < procedures.length,
      };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateAllProceduresSelection = useCallback(
    (
      db: Database,
      metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
      procedures: ProcedureMetadata[],
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      procedures.forEach((procedure) => {
        updateSimpleSelectionImpl(
          updatedMap,
          db,
          { ...metadata, procedure },
          on
        );
      });
      emit(updatedMap);
    },
    [
      selectionEnabled,
      selectedRolloutObjectMap,
      updateSimpleSelectionImpl,
      emit,
    ]
  );

  // Function selection
  const getFunctionSelectionState = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        function: FunctionMetadata;
      }
    ): SelectionState => {
      if (!selectionEnabled) return EMPTY_STATE;
      return {
        checked: selectedRolloutObjectMap.has(keyForResource(db, metadata)),
        indeterminate: false,
      };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateFunctionSelection = useCallback(
    (
      db: Database,
      metadata: {
        database: DatabaseMetadata;
        schema: SchemaMetadata;
        function: FunctionMetadata;
      },
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      updateSimpleSelectionImpl(updatedMap, db, metadata, on);
      emit(updatedMap);
    },
    [
      selectionEnabled,
      selectedRolloutObjectMap,
      updateSimpleSelectionImpl,
      emit,
    ]
  );

  const getAllFunctionsSelectionState = useCallback(
    (
      db: Database,
      metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
      functions: FunctionMetadata[]
    ): SelectionState => {
      if (!selectionEnabled || functions.length === 0) return EMPTY_STATE;
      const selected = functions.filter((func) =>
        selectedRolloutObjectMap.has(
          keyForResource(db, { ...metadata, function: func })
        )
      );
      return {
        checked: selected.length === functions.length,
        indeterminate:
          selected.length > 0 && selected.length < functions.length,
      };
    },
    [selectionEnabled, selectedRolloutObjectMap]
  );

  const updateAllFunctionsSelection = useCallback(
    (
      db: Database,
      metadata: { database: DatabaseMetadata; schema: SchemaMetadata },
      functions: FunctionMetadata[],
      on: boolean
    ) => {
      if (!selectionEnabled) return;
      const updatedMap = new Map(selectedRolloutObjectMap.entries());
      functions.forEach((func) => {
        updateSimpleSelectionImpl(
          updatedMap,
          db,
          { ...metadata, function: func },
          on
        );
      });
      emit(updatedMap);
    },
    [
      selectionEnabled,
      selectedRolloutObjectMap,
      updateSimpleSelectionImpl,
      emit,
    ]
  );

  return {
    selectionEnabled,
    getTableSelectionState,
    updateTableSelection,
    getAllTablesSelectionState,
    updateAllTablesSelection,
    getColumnSelectionState,
    updateColumnSelection,
    getAllColumnsSelectionState,
    updateAllColumnsSelection,
    getViewSelectionState,
    updateViewSelection,
    getAllViewsSelectionState,
    updateAllViewsSelection,
    getProcedureSelectionState,
    updateProcedureSelection,
    getAllProceduresSelectionState,
    updateAllProceduresSelection,
    getFunctionSelectionState,
    updateFunctionSelection,
    getAllFunctionsSelectionState,
    updateAllFunctionsSelection,
  };
}
