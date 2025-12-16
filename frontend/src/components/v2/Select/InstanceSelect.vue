<template>
  <ResourceSelect
    v-bind="$attrs"
    :remote="true"
    :loading="state.loading"
    :value="instanceName"
    :options="options"
    :custom-label="renderLabel"
    :virtual-scroll="true"
    :fallback-option="false"
    :consistent-menu-width="false"
    class="bb-instance-select"
    @search="handleSearch"
    @update:value="(val) => $emit('update:instance-name', val)"
  />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { computed, h, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useInstanceV1Store } from "@/store";
import {
  DEBOUNCE_SEARCH_DELAY,
  isValidInstanceName,
  UNKNOWN_INSTANCE_NAME,
  unknownInstance,
} from "@/types";
import { type Engine } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { getDefaultPagination, supportedEngineV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../Model/Instance";
import ResourceSelect from "./ResourceSelect.vue";

interface LocalState {
  loading: boolean;
  rawInstanceList: Instance[];
  // Track if initial fetch has been done to avoid redundant API calls
  initialized: boolean;
}

const props = withDefaults(
  defineProps<{
    instanceName?: string | undefined;
    environmentName?: string;
    projectName?: string;
    allowedEngineList?: Engine[];
    autoReset?: boolean;
  }>(),
  {
    instanceName: undefined,
    environmentName: undefined,
    allowedEngineList: () => supportedEngineV1List(),
    autoReset: true,
  }
);

const emit = defineEmits<{
  (event: "update:instance-name", value: string | undefined): void;
}>();

const { t } = useI18n();
const instanceStore = useInstanceV1Store();
const state = reactive<LocalState>({
  loading: false,
  rawInstanceList: [],
  initialized: false,
});

const initSelectedInstance = async (instanceName: string) => {
  if (isValidInstanceName(instanceName)) {
    const instance = await instanceStore.getOrFetchInstanceByName(instanceName);
    if (!state.rawInstanceList.find((ins) => ins.name === instance.name)) {
      state.rawInstanceList.unshift(instance);
    }
  }
};

const searchInstances = async (name: string) => {
  const { instances } = await instanceStore.fetchInstanceList({
    pageSize: getDefaultPagination(),
    filter: {
      engines: props.allowedEngineList,
      query: name,
      environment: props.environmentName,
      project: props.projectName,
    },
  });
  return instances;
};

const initInstanceList = async () => {
  if (props.instanceName === UNKNOWN_INSTANCE_NAME) {
    const dummyAll = {
      ...unknownInstance(),
      title: t("instance.all"),
    };
    if (!state.rawInstanceList.find((ins) => ins.name === dummyAll.name)) {
      state.rawInstanceList.unshift(dummyAll);
    }
  } else if (props.instanceName) {
    await initSelectedInstance(props.instanceName);
  }
};

const handleSearch = useDebounceFn(async (search: string) => {
  // Skip if no search term and already initialized (lazy loading optimization)
  if (!search && state.initialized) {
    return;
  }

  state.loading = true;
  try {
    const instances = await searchInstances(search);
    state.rawInstanceList = instances;
    if (!search) {
      state.initialized = true;
      await initInstanceList();
    }
  } finally {
    state.loading = false;
  }
}, DEBOUNCE_SEARCH_DELAY);

// Only fetch selected instance on mount, not the entire list.
// The full list will be fetched lazily when dropdown is opened.
// Re-initialize when filter props change.
watch(
  [
    () => props.allowedEngineList,
    () => props.environmentName,
    () => props.projectName,
  ],
  () => {
    state.initialized = false;
    state.rawInstanceList = [];
    initInstanceList();
  },
  {
    immediate: true,
  }
);

const renderLabel = (instance: Instance) => {
  if (instance.name === UNKNOWN_INSTANCE_NAME) {
    return t("instance.all");
  }
  const icon = h(InstanceV1EngineIcon, {
    instance,
    class: "bb-instance-select--engine-icon shrink-0",
  });
  const text = h("span", {}, instance.title);
  return h(
    "div",
    {
      class: "flex items-center gap-x-2",
    },
    [icon, text]
  );
};

const options = computed(() => {
  return state.rawInstanceList.map((instance) => {
    return {
      resource: instance,
      value: instance.name,
      label: instance.title,
    };
  });
});

// The instance list might change if environment changes, and the previous selected id
// might not exist in the new list. In such case, we need to reset the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset) {
    return;
  }
  if (state.loading) {
    return;
  }
  // Don't reset selection before the full instance list has been fetched
  if (!state.initialized) {
    return;
  }
  if (
    props.instanceName &&
    !state.rawInstanceList.find((item) => item.name === props.instanceName)
  ) {
    emit("update:instance-name", undefined);
  }
};

watch(
  [
    () => state.loading,
    () => props.instanceName,
    () => state.rawInstanceList,
    () => props.projectName,
  ],
  resetInvalidSelection,
  {
    immediate: true,
  }
);
</script>

<style lang="postcss" scoped>
.bb-instance-select
  :deep(.n-base-selection--active .bb-instance-select--engine-icon) {
  opacity: 0.3;
}
</style>
