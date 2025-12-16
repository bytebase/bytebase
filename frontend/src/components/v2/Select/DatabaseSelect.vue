<template>
  <ResourceSelect
    v-bind="$attrs"
    class="bb-database-select"
    :remote="true"
    :loading="state.loading"
    :multiple="multiple"
    :value="databaseName"
    :values="databaseNames"
    :options="options"
    :custom-label="renderLabel"
    @search="handleSearch"
    @update:value="(val) => $emit('update:database-name', val)"
    @update:values="(val) => $emit('update:database-names', val)"
  />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { computed, h, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useDatabaseV1Store } from "@/store";
import { workspaceNamePrefix } from "@/store/modules/v1/common";
import type { ComposedDatabase } from "@/types";
import {
  DEBOUNCE_SEARCH_DELAY,
  isValidDatabaseName,
  unknownDatabase,
} from "@/types";
import { type Engine } from "@/types/proto-es/v1/common_pb";
import {
  getDefaultPagination,
  instanceV1Name,
  supportedEngineV1List,
} from "@/utils";
import { InstanceV1EngineIcon } from "../Model";
import ResourceSelect from "./ResourceSelect.vue";

interface LocalState {
  loading: boolean;
  rawDatabaseList: ComposedDatabase[];
  // Track if initial fetch has been done to avoid redundant API calls
  initialized: boolean;
}

const props = withDefaults(
  defineProps<{
    databaseName?: string; // UNKNOWN_DATABASE_NAME stands for "ALL"
    databaseNames?: string[];
    environmentName?: string;
    projectName?: string;
    allowedEngineTypeList?: Engine[];
    includeAll?: boolean;
    autoReset?: boolean;
    filter?: (database: ComposedDatabase, index: number) => boolean;
    multiple?: boolean;
    clearable?: boolean;
    defaultSelectFirst?: boolean;
  }>(),
  {
    databaseName: undefined,
    databaseNames: undefined,
    environmentName: undefined,
    projectName: undefined,
    allowedEngineTypeList: () => supportedEngineV1List(),
    includeAll: false,
    autoReset: true,
    filter: undefined,
    multiple: false,
    clearable: false,
    defaultSelectFirst: false,
  }
);

const emit = defineEmits<{
  (event: "update:database-name", value: string | undefined): void;
  (event: "update:database-names", value: string[]): void;
}>();

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();

const state = reactive<LocalState>({
  loading: false,
  rawDatabaseList: [],
  initialized: false,
});

const initSelectedDatabases = async (databaseNames: string[]) => {
  for (const databaseName of databaseNames) {
    if (isValidDatabaseName(databaseName)) {
      const db = await databaseStore.getOrFetchDatabaseByName(databaseName);
      if (!state.rawDatabaseList.find((d) => d.name === db.name)) {
        state.rawDatabaseList.unshift(db);
      }
    }
  }
};

const searchDatabases = async (name: string) => {
  const { databases } = await databaseStore.fetchDatabases({
    parent: props.projectName ?? `${workspaceNamePrefix}-`,
    filter: {
      environment: props.environmentName,
      engines: props.allowedEngineTypeList,
      query: name,
    },
    pageSize: getDefaultPagination(),
  });
  return databases;
};

const initDatabaseList = async () => {
  if (props.includeAll) {
    const dummyAll = {
      ...unknownDatabase(),
      databaseName: t("database.all"),
    };
    if (!state.rawDatabaseList.find((d) => d.name === dummyAll.name)) {
      state.rawDatabaseList.unshift(dummyAll);
    }
  }
  if (props.databaseName) {
    await initSelectedDatabases([props.databaseName]);
  }
  if (props.databaseNames) {
    await initSelectedDatabases(props.databaseNames);
  }
};

const handleSearch = useDebounceFn(async (search: string) => {
  // Skip if no search term and already initialized (lazy loading optimization)
  if (!search && state.initialized) {
    return;
  }

  state.loading = true;
  try {
    const databases = await searchDatabases(search);
    state.rawDatabaseList = databases;
    if (!search) {
      state.initialized = true;
      await initDatabaseList();
    }
  } finally {
    state.loading = false;
  }
}, DEBOUNCE_SEARCH_DELAY);

// Only fetch selected database(s) on mount, not the entire list.
// The full list will be fetched lazily when dropdown is opened.
// Re-initialize when filter props change.
watch(
  () => [props.environmentName, props.allowedEngineTypeList],
  () => {
    state.initialized = false;
    state.rawDatabaseList = [];
    initDatabaseList();
  },
  {
    deep: true,
    immediate: true,
  }
);

const combinedDatabaseList = computed(() => {
  if (props.filter) {
    return state.rawDatabaseList.filter(props.filter);
  }

  return state.rawDatabaseList;
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

watch(
  () => options.value,
  () => {
    if (!props.defaultSelectFirst || props.multiple) {
      return;
    }
    if (options.value.length === 0) {
      return;
    }

    emit("update:database-name", options.value[0].value);
  },
  { immediate: true }
);

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
  if (!props.autoReset || props.multiple) return;
  if (state.loading) return;
  // Don't reset selection before the full database list has been fetched
  if (!state.initialized) return;
  if (
    props.databaseName &&
    !combinedDatabaseList.value.find((item) => item.name === props.databaseName)
  ) {
    emit("update:database-name", undefined);
  }
};

watch(() => combinedDatabaseList.value, resetInvalidSelection, {
  immediate: true,
  deep: true,
});
</script>
