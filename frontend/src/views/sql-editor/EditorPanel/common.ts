import { pausableWatch, useElementSize } from "@vueuse/core";
import { first } from "lodash-es";
import type { DataTableInst, SelectOption } from "naive-ui";
import { computed, ref, unref, watch, type MaybeRef } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { hasSchemaProperty, nextAnimationFrame } from "@/utils";
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

export type AutoHeightDataTableOptions = {
  maxHeight: number | null;
};
const defaultAutoHeightDataTableOptions = (): AutoHeightDataTableOptions => ({
  maxHeight: null,
});

export const useAutoHeightDataTable = (
  data?: MaybeRef<unknown[]>,
  options?: MaybeRef<Partial<AutoHeightDataTableOptions>>
) => {
  const opts = computed(() => ({
    ...defaultAutoHeightDataTableOptions(),
    ...unref(options),
  }));
  const GAP_HEIGHT = 2;
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
  const tableBodyHeight = ref(0);

  const { pause: pauseWatch, resume: resumeWatch } = pausableWatch(
    [() => unref(data), containerHeight, tableHeaderHeight, opts],
    async () => {
      if (containerHeight.value === 0 || tableHeaderHeight.value === 0) return;
      if (opts.value.maxHeight === null) {
        // The table height is always limited by the container height
        // and need not to be adjusted automatically
        tableBodyHeight.value =
          containerHeight.value - tableHeaderHeight.value - GAP_HEIGHT;
        return;
      }

      pauseWatch();
      const experimentalMaxBodyHeight =
        opts.value.maxHeight - tableHeaderHeight.value - GAP_HEIGHT;
      tableBodyHeight.value = experimentalMaxBodyHeight;
      await nextAnimationFrame();
      resumeWatch();
    },
    { immediate: true }
  );

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
