<template>
  <NSelect
    :value="combinedValue"
    :options="options"
    :placeholder="placeholder ?? $t('database.select')"
    :virtual-scroll="true"
    :multiple="multiple"
    :filter="filterByDatabaseName"
    :filterable="true"
    :clearable="clearable"
    class="bb-database-select"
    style="width: 12rem"
    v-bind="$attrs"
    :render-label="renderLabel"
    @update:value="handleValueUpdated"
  />
</template>

<script lang="ts" setup>
import type { SelectOption, SelectRenderLabel } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed, h, watch } from "vue";
import { useSlots } from "vue";
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

interface DatabaseSelectOption extends SelectOption {
  value: string;
  database: ComposedDatabase;
}

const slots = useSlots();
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
    placeholder?: string;
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
    placeholder: undefined,
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

const combinedValue = computed(() => {
  if (props.multiple) {
    return props.databaseNames || [];
  } else {
    return props.databaseName;
  }
});

const handleValueUpdated = (value: string | string[]) => {
  if (props.multiple) {
    if (!value) {
      // normalize value
      value = [];
    }
    emit("update:database-names", value as string[]);
  } else {
    if (value === null) {
      // normalize value
      value = "";
    }
    emit("update:database-name", value as string);
  }
};

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
  return combinedDatabaseList.value.map<DatabaseSelectOption>((database) => {
    return {
      database,
      value: database.name,
      label: database.databaseName,
    };
  });
});

const renderLabel: SelectRenderLabel = (option) => {
  const { database } = option as DatabaseSelectOption;
  if (!database) {
    return;
  }

  if (slots.default) {
    return slots.default({ database });
  }

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

const filterByDatabaseName = (pattern: string, option: SelectOption) => {
  const { database } = option as DatabaseSelectOption;
  return database.databaseName.toLowerCase().includes(pattern.toLowerCase());
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
