<template>
  <NSelect
    :value="database"
    :options="options"
    :placeholder="$t('database.select')"
    :virtual-scroll="true"
    :filter="filterByName"
    :filterable="true"
    style="width: 12rem"
    @update:value="$emit('update:database', $event)"
  />
</template>

<script lang="ts" setup>
import { computed, watch, watchEffect } from "vue";
import { NSelect, SelectOption } from "naive-ui";
import { useI18n } from "vue-i18n";

import { useCurrentUser, useDatabaseStore } from "@/store";
import {
  Database,
  DatabaseId,
  EngineType,
  EngineTypeList,
  InstanceId,
  ProjectId,
  UNKNOWN_ID,
  unknown,
} from "@/types";

interface DatabaseSelectOption extends SelectOption {
  value: DatabaseId;
  database: Database;
}

const props = withDefaults(
  defineProps<{
    database: DatabaseId | undefined;
    environment?: string;
    instance?: InstanceId;
    project?: ProjectId;
    allowedEngineTypeList?: readonly EngineType[];
    includeAll?: boolean;
    autoReset?: boolean;
    filter?: (database: Database, index: number) => boolean;
  }>(),
  {
    environment: undefined,
    instance: undefined,
    project: undefined,
    allowedEngineTypeList: () => EngineTypeList,
    includeAll: false,
    autoReset: true,
    filter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:database", value: DatabaseId | undefined): void;
}>();

const { t } = useI18n();
const currentUser = useCurrentUser();
const databaseStore = useDatabaseStore();

const prepare = () => {
  databaseStore.fetchDatabaseList();
};
watchEffect(prepare);

const rawDatabaseList = computed(() => {
  const list = databaseStore.getDatabaseListByPrincipalId(currentUser.value.id);

  return list.filter((db) => {
    if (props.environment && props.environment !== String(UNKNOWN_ID)) {
      if (db.instance.environment.id !== props.environment) return false;
    }
    if (props.instance && props.instance !== UNKNOWN_ID) {
      if (db.instance.id !== props.instance) return false;
    }
    if (props.project && props.project !== UNKNOWN_ID) {
      if (db.project.id !== props.project) return false;
    }
    if (!props.allowedEngineTypeList.includes(db.instance.engine)) return false;

    return true;
  });
});

const combinedDatabaseList = computed(() => {
  let list = [...rawDatabaseList.value];

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (props.database === UNKNOWN_ID || props.includeAll) {
    const dummyAll = unknown("DATABASE");
    dummyAll.name = t("database.all");
    list.unshift(dummyAll);
  }

  return list;
});

const options = computed(() => {
  return combinedDatabaseList.value.map<DatabaseSelectOption>((database) => {
    return {
      database,
      value: database.id,
      label: database.name,
    };
  });
});

const filterByName = (pattern: string, option: SelectOption) => {
  const { database } = option as DatabaseSelectOption;
  return database.name.toLowerCase().includes(pattern.toLowerCase());
};

// The database list might change if environment changes, and the previous selected id
// might not exist in the new list. In such case, we need to invalidate the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset) return;
  if (
    props.database &&
    !combinedDatabaseList.value.find((item) => item.id === props.database)
  ) {
    emit("update:database", undefined);
  }
};

watch([() => props.database, combinedDatabaseList], resetInvalidSelection, {
  immediate: true,
});
</script>
