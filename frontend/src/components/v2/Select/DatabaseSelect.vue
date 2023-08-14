<template>
  <NSelect
    :value="database"
    :options="options"
    :placeholder="$t('database.select')"
    :virtual-scroll="true"
    :filter="filterByDatabaseName"
    :filterable="true"
    style="width: 12rem"
    v-bind="$attrs"
    :render-label="renderLabel"
    @update:value="$emit('update:database', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect, SelectOption, SelectRenderLabel } from "naive-ui";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  useCurrentUserV1,
  useSearchDatabaseV1List,
  useDatabaseV1Store,
} from "@/store";
import { ComposedDatabase, UNKNOWN_ID, unknownDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { instanceV1Name, supportedEngineV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../Model";

interface DatabaseSelectOption extends SelectOption {
  value: string;
  database: ComposedDatabase;
}

const props = withDefaults(
  defineProps<{
    database: string | undefined;
    environment?: string;
    instance?: string;
    project?: string;
    allowedEngineTypeList?: readonly Engine[];
    includeAll?: boolean;
    autoReset?: boolean;
    filter?: (database: ComposedDatabase, index: number) => boolean;
  }>(),
  {
    environment: undefined,
    instance: undefined,
    project: undefined,
    allowedEngineTypeList: () => supportedEngineV1List(),
    includeAll: false,
    autoReset: true,
    filter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:database", value: string | undefined): void;
}>();

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const { ready } = useSearchDatabaseV1List({
  parent: "instances/-",
});

const rawDatabaseList = computed(() => {
  const list = useDatabaseV1Store().databaseListByUser(currentUserV1.value);

  return list.filter((db) => {
    if (props.environment && props.environment !== String(UNKNOWN_ID)) {
      if (db.effectiveEnvironmentEntity.uid !== props.environment) {
        return false;
      }
    }
    if (props.instance && props.instance !== String(UNKNOWN_ID)) {
      if (db.instanceEntity.uid !== props.instance) return false;
    }
    if (props.project && props.project !== String(UNKNOWN_ID)) {
      if (db.projectEntity.uid !== props.project) return false;
    }
    if (!props.allowedEngineTypeList.includes(db.instanceEntity.engine)) {
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

  if (props.database === String(UNKNOWN_ID) || props.includeAll) {
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
      value: database.uid,
      label: database.databaseName,
    };
  });
});

const renderLabel: SelectRenderLabel = (option) => {
  const { database } = option as DatabaseSelectOption;
  return h(
    "div",
    {
      class: "flex flex-row justify-start items-center",
    },
    [
      h(InstanceV1EngineIcon, {
        class: "mr-1",
        instance: database.instanceEntity,
      }),
      h(
        "div",
        {
          class: "mr-1",
        },
        [database.databaseName]
      ),
      h(
        "div",
        {
          class: "text-xs opacity-60",
        },
        [`(${instanceV1Name(database.instanceEntity)})`]
      ),
    ]
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
    props.database &&
    !combinedDatabaseList.value.find((item) => item.uid === props.database)
  ) {
    emit("update:database", undefined);
  }
};

watch([() => props.database, combinedDatabaseList], resetInvalidSelection, {
  immediate: true,
});
</script>
