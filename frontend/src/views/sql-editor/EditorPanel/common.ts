import { useElementSize } from "@vueuse/core";
import { first } from "lodash-es";
import type { DataTableInst, SelectOption } from "naive-ui";
import { computed, ref, watch } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { hasSchemaProperty } from "@/utils";
import { useEditorPanelContext } from "./context";

export const useSelectSchema = () => {
  const { database, instance } = useConnectionOfCurrentSQLEditorTab();
  const { selectedSchemaName } = useEditorPanelContext();
  const databaseMetadata = computed(() => {
    return useDBSchemaV1Store().getDatabaseMetadata(
      database.value.name,
      DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
    );
  });
  const options = computed(() => {
    return databaseMetadata.value.schemas.map<SelectOption>((schema) => ({
      label: schema.name,
      value: schema.name,
    }));
  });
  const showSchemaSelect = computed(() => {
    return hasSchemaProperty(instance.value.engine);
  });

  watch(
    [databaseMetadata, selectedSchemaName],
    ([database, schema]) => {
      if (database && schema === undefined) {
        selectedSchemaName.value = first(database.schemas)?.name;
      }
    },
    { immediate: true }
  );

  return { selectedSchemaName, showSchemaSelect, options };
};

export const useAutoHeightDataTable = () => {
  const dataTableRef = ref<DataTableInst>();
  const containerElRef = ref<HTMLElement>();
  const tableHeaderElRef = computed(
    () =>
      containerElRef.value?.querySelector("thead.n-data-table-thead") as
        | HTMLElement
        | undefined
  );
  const { height: containerHeight } = useElementSize(containerElRef);
  const { height: tableHeaderHeight } = useElementSize(tableHeaderElRef);
  const tableBodyHeight = computed(() => {
    return containerHeight.value - tableHeaderHeight.value - 2;
  });
  // Use this to avoid unnecessary initial rendering
  const layoutReady = computed(() => tableHeaderHeight.value > 0);

  return {
    dataTableRef,
    containerElRef,
    tableHeaderElRef,
    tableBodyHeight,
    layoutReady,
  };
};
