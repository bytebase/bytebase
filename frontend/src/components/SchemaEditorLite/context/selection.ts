import { pick } from "lodash-es";
import type { Ref } from "vue";
import { computed } from "vue";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { RolloutObject } from "../types";
import type { SchemaEditorEvents } from ".";
import { keyForResource } from "./common";

export const useSelection = (
  selectedRolloutObjects: Ref<RolloutObject[] | undefined>,
  events: SchemaEditorEvents
) => {
  const selectionEnabled = computed(() => {
    return selectedRolloutObjects.value !== undefined;
  });

  const selectedRolloutObjectMap = computed(() => {
    return new Map(
      selectedRolloutObjects.value?.map((ro) => {
        const key = keyForResource(ro.db, ro.metadata);
        return [key, ro];
      })
    );
  });

  const updateTableSelectionImpl = (
    map: Map<string, RolloutObject>,
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    on: boolean
  ) => {
    const { table } = metadata;
    if (on) {
      // select the table itself including all its columns
      map.set(keyForResource(db, metadata), {
        db,
        metadata,
      });
      table.columns.forEach((column) => {
        map.set(keyForResource(db, { ...metadata, column }), {
          db,
          metadata: { ...metadata, column },
        });
      });
    } else {
      // de-select the table it self including all its columns
      map.delete(keyForResource(db, metadata));
      table.columns.forEach((column) => {
        map.delete(keyForResource(db, { ...metadata, column }));
      });
    }
  };
  const updateColumnSelectionImpl = (
    map: Map<string, RolloutObject>,
    db: ComposedDatabase,
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
      // ensure the column's table is selected
      map.set(keyForResource(db, tableMeta), {
        db,
        metadata: tableMeta,
      });
      // then select the column itself
      map.set(keyForResource(db, metadata), {
        db,
        metadata,
      });
    } else {
      // de-select the column
      map.delete(keyForResource(db, metadata));
    }
  };
  const updateViewSelectionImpl = (
    map: Map<string, RolloutObject>,
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      view: ViewMetadata;
    },
    on: boolean
  ) => {
    const key = keyForResource(db, metadata);
    if (on) {
      // select the view
      map.set(key, {
        db,
        metadata,
      });
    } else {
      // de-select the view
      map.delete(key);
    }
  };
  const updateProcedureSelectionImpl = (
    map: Map<string, RolloutObject>,
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      procedure: ProcedureMetadata;
    },
    on: boolean
  ) => {
    const key = keyForResource(db, metadata);
    if (on) {
      // select the procedure
      map.set(key, {
        db,
        metadata,
      });
    } else {
      // de-select the procedure
      map.delete(key);
    }
  };
  const updateFunctionSelectionImpl = (
    map: Map<string, RolloutObject>,
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      function: FunctionMetadata;
    },
    on: boolean
  ) => {
    const key = keyForResource(db, metadata);
    if (on) {
      // select the function
      map.set(key, {
        db,
        metadata,
      });
    } else {
      // de-select the function
      map.delete(key);
    }
  };
  const emit = (map: Map<string, RolloutObject>) => {
    events.emit("update:selected-rollout-objects", Array.from(map.values()));
  };

  const getTableSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    }
  ) => {
    if (!selectionEnabled.value) {
      return { checked: false, indeterminate: false };
    }

    const { table } = metadata;
    const tableChecked = selectedRolloutObjectMap.value.has(
      keyForResource(db, metadata)
    );
    if (table.columns.length === 0) {
      return {
        checked: tableChecked,
        indeterminate: false,
      };
    }

    const checkedColumns = table.columns.filter((column) => {
      const columnChecked = selectedRolloutObjectMap.value.has(
        keyForResource(db, {
          ...metadata,
          column,
        })
      );
      return !!columnChecked;
    });

    const state = {
      checked: checkedColumns.length === table.columns.length,
      indeterminate: false,
    };
    if (!state.checked) {
      if (tableChecked || checkedColumns.length > 0) {
        state.indeterminate = true;
      }
    }
    return state;
  };
  const updateTableSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    on: boolean
  ) => {
    if (!selectionEnabled.value) return;
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    updateTableSelectionImpl(updatedMap, db, metadata, on);
    emit(updatedMap);
  };
  const getAllTablesSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
    },
    tables: TableMetadata[]
  ) => {
    if (!selectionEnabled.value || tables.length === 0) {
      return { checked: false, indeterminate: false };
    }
    const selectedTables = tables.filter((table) => {
      return selectedRolloutObjectMap.value.has(
        keyForResource(db, {
          ...metadata,
          table,
        })
      );
    });
    return {
      checked: selectedTables.length === tables.length,
      indeterminate:
        selectedTables.length > 0 && selectedTables.length < tables.length,
    };
  };
  const updateAllTablesSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
    },
    tables: TableMetadata[],
    on: boolean
  ) => {
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    tables.forEach((table) => {
      updateTableSelectionImpl(
        updatedMap,
        db,
        {
          ...metadata,
          table,
        },
        on
      );
    });

    emit(updatedMap);
  };
  const getColumnSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      column: ColumnMetadata;
    }
  ) => {
    if (!selectionEnabled.value) {
      return { checked: false, indeterminate: false };
    }

    const checked = selectedRolloutObjectMap.value.has(
      keyForResource(db, metadata)
    );
    return { checked, indeterminate: false };
  };
  const updateColumnSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
      column: ColumnMetadata;
    },
    on: boolean
  ) => {
    if (!selectionEnabled.value) return;
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    updateColumnSelectionImpl(updatedMap, db, metadata, on);
    emit(updatedMap);
  };
  const getAllColumnsSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    columns: ColumnMetadata[]
  ) => {
    if (!selectionEnabled.value || columns.length === 0) {
      return { checked: false, indeterminate: false };
    }
    const selectedColumns = columns.filter((column) => {
      return selectedRolloutObjectMap.value.has(
        keyForResource(db, {
          ...metadata,
          column,
        })
      );
    });
    return {
      checked: selectedColumns.length === columns.length,
      indeterminate:
        selectedColumns.length > 0 && selectedColumns.length < columns.length,
    };
  };
  const updateAllColumnsSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      table: TableMetadata;
    },
    columns: ColumnMetadata[],
    on: boolean
  ) => {
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    columns.forEach((column) => {
      updateColumnSelectionImpl(
        updatedMap,
        db,
        {
          ...metadata,
          column,
        },
        on
      );
    });

    emit(updatedMap);
  };

  const getViewSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      view: ViewMetadata;
    }
  ) => {
    if (!selectionEnabled.value) {
      return { checked: false, indeterminate: false };
    }

    const checked = selectedRolloutObjectMap.value.has(
      keyForResource(db, metadata)
    );
    return {
      checked,
      indeterminate: false,
    };
  };
  const updateViewSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      view: ViewMetadata;
    },
    on: boolean
  ) => {
    if (!selectionEnabled.value) return;
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    updateViewSelectionImpl(updatedMap, db, metadata, on);
    emit(updatedMap);
  };
  const getAllViewsSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
    },
    views: ViewMetadata[]
  ) => {
    if (!selectionEnabled.value || views.length === 0) {
      return { checked: false, indeterminate: false };
    }
    const selected = views.filter((view) => {
      return selectedRolloutObjectMap.value.has(
        keyForResource(db, {
          ...metadata,
          view,
        })
      );
    });
    return {
      checked: selected.length === views.length,
      indeterminate: selected.length > 0 && selected.length < views.length,
    };
  };
  const updateAllViewsSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
    },
    views: ViewMetadata[],
    on: boolean
  ) => {
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    views.forEach((view) => {
      updateViewSelectionImpl(
        updatedMap,
        db,
        {
          ...metadata,
          view,
        },
        on
      );
    });

    emit(updatedMap);
  };

  const getProcedureSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      procedure: ProcedureMetadata;
    }
  ) => {
    if (!selectionEnabled.value) {
      return { checked: false, indeterminate: false };
    }

    const checked = selectedRolloutObjectMap.value.has(
      keyForResource(db, metadata)
    );
    return {
      checked,
      indeterminate: false,
    };
  };
  const updateProcedureSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      procedure: ProcedureMetadata;
    },
    on: boolean
  ) => {
    if (!selectionEnabled.value) return;
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    updateProcedureSelectionImpl(updatedMap, db, metadata, on);
    emit(updatedMap);
  };
  const getAllProceduresSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
    },
    procedures: ProcedureMetadata[]
  ) => {
    if (!selectionEnabled.value || procedures.length === 0) {
      return { checked: false, indeterminate: false };
    }
    const selected = procedures.filter((procedure) => {
      return selectedRolloutObjectMap.value.has(
        keyForResource(db, {
          ...metadata,
          procedure,
        })
      );
    });
    return {
      checked: selected.length === procedures.length,
      indeterminate: selected.length > 0 && selected.length < procedures.length,
    };
  };
  const updateAllProceduresSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
    },
    procedures: ProcedureMetadata[],
    on: boolean
  ) => {
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    procedures.forEach((procedure) => {
      updateProcedureSelectionImpl(
        updatedMap,
        db,
        {
          ...metadata,
          procedure,
        },
        on
      );
    });

    emit(updatedMap);
  };

  const getFunctionSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      function: FunctionMetadata;
    }
  ) => {
    if (!selectionEnabled.value) {
      return { checked: false, indeterminate: false };
    }

    const checked = selectedRolloutObjectMap.value.has(
      keyForResource(db, metadata)
    );
    return {
      checked,
      indeterminate: false,
    };
  };
  const updateFunctionSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
      function: FunctionMetadata;
    },
    on: boolean
  ) => {
    if (!selectionEnabled.value) return;
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    updateFunctionSelectionImpl(updatedMap, db, metadata, on);
    emit(updatedMap);
  };
  const getAllFunctionsSelectionState = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
    },
    functions: FunctionMetadata[]
  ) => {
    if (!selectionEnabled.value || functions.length === 0) {
      return { checked: false, indeterminate: false };
    }
    const selected = functions.filter((func) => {
      return selectedRolloutObjectMap.value.has(
        keyForResource(db, {
          ...metadata,
          function: func,
        })
      );
    });
    return {
      checked: selected.length === functions.length,
      indeterminate: selected.length > 0 && selected.length < functions.length,
    };
  };
  const updateAllFunctionsSelection = (
    db: ComposedDatabase,
    metadata: {
      database: DatabaseMetadata;
      schema: SchemaMetadata;
    },
    functions: FunctionMetadata[],
    on: boolean
  ) => {
    const updatedMap = new Map(selectedRolloutObjectMap.value.entries());
    functions.forEach((func) => {
      updateFunctionSelectionImpl(
        updatedMap,
        db,
        {
          ...metadata,
          function: func,
        },
        on
      );
    });

    emit(updatedMap);
  };

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
};
