import { pick } from "lodash-es";
import { Ref, computed } from "vue";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaEditorEvents } from ".";
import { RolloutObject } from "../types";
import { keyForResource } from "./common";

export const useSelection = (
  selectedRolloutObjects: Ref<RolloutObject[] | undefined>,
  events: SchemaEditorEvents
) => {
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
    if (!selectedRolloutObjects.value) {
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
    if (!selectedRolloutObjects.value) return;
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
    if (!selectedRolloutObjects.value || tables.length === 0) {
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
    if (!selectedRolloutObjects.value) {
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
    if (!selectedRolloutObjects.value) return;
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
    if (!selectedRolloutObjects.value || columns.length === 0) {
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

  return {
    getTableSelectionState,
    updateTableSelection,
    getAllTablesSelectionState,
    updateAllTablesSelection,
    getColumnSelectionState,
    updateColumnSelection,
    getAllColumnsSelectionState,
    updateAllColumnsSelection,
  };
};
