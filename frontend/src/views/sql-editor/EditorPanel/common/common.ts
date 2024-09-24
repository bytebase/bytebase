import { first } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { computed, watch } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { hasSchemaProperty } from "@/utils";
import { useEditorPanelContext } from "../context";

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
