import { first } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { computed, watch } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import { hasSchemaProperty } from "@/utils";
import { convertEngineToNew } from "@/utils/v1/common-conversions";
import { useCurrentTabViewStateContext } from "../context/viewState";

export const useSelectSchema = () => {
  const { database, instance } = useConnectionOfCurrentSQLEditorTab();
  const { selectedSchemaName } = useCurrentTabViewStateContext();
  const databaseMetadata = computed(() => {
    return useDBSchemaV1Store().getDatabaseMetadata(database.value.name);
  });
  const options = computed(() => {
    return databaseMetadata.value.schemas.map<SelectOption>((schema) => ({
      label: schema.name,
      value: schema.name,
    }));
  });
  const showSchemaSelect = computed(() => {
    return hasSchemaProperty(convertEngineToNew(instance.value.engine));
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
