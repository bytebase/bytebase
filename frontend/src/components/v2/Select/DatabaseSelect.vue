<template>
  <ResourceSelect
    v-bind="$attrs"
    class="bb-database-select"
    :placeholder="$t('database.select')"
    :multiple="multiple"
    :value="databaseName"
    :values="databaseNames"
    :options="options"
    :custom-label="renderLabel"
    @update:value="(val) => $emit('update:database-name', val)"
    @update:values="(val) => $emit('update:database-names', val)"
  />
</template>

<script lang="ts" setup>
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useDatabaseV1Store } from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { ComposedDatabase } from "@/types";
import {
  isValidDatabaseName,
  isValidEnvironmentName,
  isValidInstanceName,
  isValidProjectName,
  unknownDatabase,
} from "@/types";
import type { Engine } from "@/types/proto/v1/common";
import { instanceV1Name, supportedEngineV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../Model";
import ResourceSelect from "./ResourceSelect.vue";

const props = withDefaults(
  defineProps<{
    databaseName?: string; // UNKNOWN_DATABASE_NAME stands for "ALL"
    databaseNames?: string[];
    environmentName?: string;
    instanceName?: string;
    projectName?: string;
    allowedEngineTypeList?: readonly Engine[];
    includeAll?: boolean;
    autoReset?: boolean;
    filter?: (database: ComposedDatabase, index: number) => boolean;
    multiple?: boolean;
    clearable?: boolean;
  }>(),
  {
    databaseName: undefined,
    databaseNames: undefined,
    environmentName: undefined,
    instanceName: undefined,
    projectName: undefined,
    allowedEngineTypeList: () => supportedEngineV1List(),
    includeAll: false,
    autoReset: true,
    filter: undefined,
    multiple: false,
    clearable: false,
  }
);

const emit = defineEmits<{
  (event: "update:database-name", value: string | undefined): void;
  (event: "update:database-names", value: string[]): void;
}>();

const { t } = useI18n();
const { ready } = useDatabaseV1List(props.projectName || props.instanceName);

const rawDatabaseList = computed(() => {
  const list = useDatabaseV1Store().databaseListByUser;

  return list.filter((db) => {
    if (
      isValidEnvironmentName(props.environmentName) &&
      db.effectiveEnvironment !== props.environmentName
    ) {
      return false;
    }
    if (
      isValidInstanceName(props.instanceName) &&
      props.instanceName !== db.instance
    ) {
      return false;
    }
    if (
      isValidProjectName(props.projectName) &&
      db.project !== props.projectName
    ) {
      return false;
    }
    if (!props.allowedEngineTypeList.includes(db.instanceResource.engine)) {
      return false;
    }

    return true;
  });
});

const combinedDatabaseList = computed(() => {
  let list = [...rawDatabaseList.value];

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (props.includeAll) {
    const dummyAll = {
      ...unknownDatabase(),
      databaseName: t("database.all"),
    };
    list.unshift(dummyAll);
  }

  return list;
});

const options = computed(() => {
  return combinedDatabaseList.value.map((database) => {
    return {
      resource: database,
      value: database.name,
      label: database.databaseName,
    };
  });
});

const renderLabel = (database: ComposedDatabase) => {
  const children = [h("div", {}, [database.databaseName])];
  if (isValidDatabaseName(database.name)) {
    // prefix engine icon
    children.unshift(
      h(InstanceV1EngineIcon, {
        class: "mr-1",
        instance: database.instanceResource,
      })
    );
    // suffix engine name
    children.push(
      h(
        "div",
        {
          class: "text-xs opacity-60 ml-1",
        },
        [`(${instanceV1Name(database.instanceResource)})`]
      )
    );
  }
  return h(
    "div",
    {
      class: "w-full flex flex-row justify-start items-center truncate",
    },
    children
  );
};

// The database list might change if environment changes, and the previous selected id
// might not exist in the new list. In such case, we need to invalidate the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset) return;
  if (
    ready.value &&
    props.databaseName &&
    !combinedDatabaseList.value.find((item) => item.name === props.databaseName)
  ) {
    emit("update:database-name", undefined);
  }
};

watch(
  [
    () => [props.projectName, props.environmentName, props.databaseName],
    combinedDatabaseList,
  ],
  resetInvalidSelection,
  {
    immediate: true,
  }
);
</script>
