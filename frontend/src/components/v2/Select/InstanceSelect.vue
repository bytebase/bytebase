<template>
  <ResourceSelect
    v-bind="$attrs"
    :value="instanceName"
    :options="options"
    :placeholder="$t('instance.select')"
    :custom-label="renderLabel"
    :virtual-scroll="true"
    :fallback-option="false"
    :consistent-menu-width="false"
    class="bb-instance-select"
    @update:value="(val) => $emit('update:instance-name', val)"
  />
</template>

<script lang="ts" setup>
import { computed, h, watch, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useInstanceResourceList, useInstanceV1Store } from "@/store";
import {
  UNKNOWN_INSTANCE_NAME,
  isValidEnvironmentName,
  isValidInstanceName,
  unknownInstance,
  type ComposedInstance,
} from "@/types";
import { type Engine, engineToJSON } from "@/types/proto/v1/common";
import type { InstanceResource } from "@/types/proto/v1/instance_service";
import { supportedEngineV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../Model/Instance";
import ResourceSelect from "./ResourceSelect.vue";

interface LocalState {
  loading: boolean;
  rawInstanceList: ComposedInstance[];
}

const props = withDefaults(
  defineProps<{
    instanceName?: string | undefined;
    environmentName?: string;
    allowedEngineList?: readonly Engine[];
    includeAll?: boolean;
    autoReset?: boolean;
    filter?: (instance: InstanceResource, index: number) => boolean;
  }>(),
  {
    instanceName: undefined,
    environmentName: undefined,
    allowedEngineList: () => supportedEngineV1List(),
    includeAll: false,
    autoReset: true,
    filter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:instance-name", value: string | undefined): void;
}>();

const { t } = useI18n();
const instanceList = useInstanceResourceList();
const instanceStore = useInstanceV1Store();

const state = reactive<LocalState>({
  loading: true,
  rawInstanceList: [],
});

const rawInstanceList = computed(() => {
  let list = [...instanceList.value];
  if (isValidEnvironmentName(props.environmentName)) {
    list = instanceList.value.filter(
      (instance) => instance.environment === props.environmentName
    );
  }
  // Filter by engine type
  list = list.filter((instance) =>
    props.allowedEngineList.includes(instance.engine)
  );
  return list;
});

const initSelectedInstance = async (instanceName: string[]) => {
  if (isValidInstanceName(instanceName)) {
    const instance = await instanceStore.getOrFetchInstanceByName(instanceName);
    if (!state.rawInstanceList.find((ins) => ins.name === instance.name)) {
      state.rawInstanceList.unshift(instance);
    }
  }
};

const filterParams = computed(() => {
  const list = [];
  if (isValidEnvironmentName(props.environmentName)) {
    list.push(`environment == "${props.environmentName}"`);
  }
  if (props.allowedEngineList.length > 0) {
    list.push(
      `engine in [${props.allowedEngineList.map((e) => `"${engineToJSON(e)}"`).join(", ")}]`
    );
  }

  return list;
});

const combinedInstanceList = computed(() => {
  let list = rawInstanceList.value;

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (props.instanceName === UNKNOWN_INSTANCE_NAME || props.includeAll) {
    const dummyAll = {
      ...unknownInstance(),
      title: t("instance.all"),
    };
    list.unshift(dummyAll);
  }

  return list;
});

const renderLabel = (instance: InstanceResource) => {
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
  return combinedInstanceList.value.map((instance) => {
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
  if (!props.autoReset) return;
  if (
    props.instanceName &&
    !combinedInstanceList.value.find((item) => item.name === props.instanceName)
  ) {
    emit("update:instance-name", undefined);
  }
};

watch([() => props.instanceName, combinedInstanceList], resetInvalidSelection, {
  immediate: true,
});
</script>

<style lang="postcss" scoped>
.bb-instance-select
  :deep(.n-base-selection--active .bb-instance-select--engine-icon) {
  opacity: 0.3;
}
</style>
