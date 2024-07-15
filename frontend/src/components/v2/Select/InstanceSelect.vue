<template>
  <NSelect
    v-bind="$attrs"
    :value="instanceName"
    :options="options"
    :placeholder="$t('instance.select')"
    :render-label="renderLabel"
    :filter="filterByTitle"
    :filterable="true"
    :virtual-scroll="true"
    :fallback-option="false"
    :consistent-menu-width="false"
    class="bb-instance-select"
    @update:value="$emit('update:instance-name', $event)"
  />
</template>

<script lang="ts" setup>
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed, watch, h } from "vue";
import { useI18n } from "vue-i18n";
import { useInstanceV1List } from "@/store";
import type { ComposedInstance } from "@/types";
import {
  UNKNOWN_INSTANCE_NAME,
  isValidEnvironmentName,
  unknownInstance,
} from "@/types";
import type { Engine } from "@/types/proto/v1/common";
import { State } from "@/types/proto/v1/common";
import { supportedEngineV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../Model/Instance";

interface InstanceSelectOption extends SelectOption {
  value: string;
  instance: ComposedInstance;
}

const props = withDefaults(
  defineProps<{
    instanceName?: string | undefined;
    environmentName?: string;
    allowedEngineList?: readonly Engine[];
    includeAll?: boolean;
    includeArchived?: boolean;
    autoReset?: boolean;
    useResourceId?: boolean;
    filter?: (instance: ComposedInstance, index: number) => boolean;
  }>(),
  {
    instanceName: undefined,
    environmentName: undefined,
    allowedEngineList: () => supportedEngineV1List(),
    includeAll: false,
    includeArchived: false,
    autoReset: true,
    useResourceId: false,
    filter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:instance-name", value: string | undefined): void;
}>();

const { t } = useI18n();
const { instanceList: allInstanceList, ready } = useInstanceV1List(
  true /* showDeleted */
);

const rawInstanceList = computed(() => {
  let list = [...allInstanceList.value];
  if (isValidEnvironmentName(props.environmentName)) {
    list = allInstanceList.value.filter(
      (instance) => instance.environment === props.environmentName
    );
  }
  // Filter by engine type
  list = list.filter((instance) =>
    props.allowedEngineList.includes(instance.engine)
  );
  return list;
});

const combinedInstanceList = computed(() => {
  let list = rawInstanceList.value.filter((instance) => {
    if (props.includeArchived) return true;
    if (instance.state === State.ACTIVE) return true;
    // ARCHIVED
    if (instance.name === props.instanceName) return true;
    return false;
  });

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

const renderLabel = (option: SelectOption) => {
  const { instance } = option as InstanceSelectOption;
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
  return combinedInstanceList.value.map<InstanceSelectOption>((instance) => {
    return {
      instance,
      value: instance.name,
      label: instance.title,
    };
  });
});

const filterByTitle = (pattern: string, option: SelectOption) => {
  const { instance } = option as InstanceSelectOption;
  return instance.title.toLowerCase().includes(pattern.toLowerCase());
};

// The instance list might change if environment changes, and the previous selected id
// might not exist in the new list. In such case, we need to reset the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset) return;
  if (
    ready.value &&
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
