<template>
  <NSelect
    :value="instance"
    :options="options"
    :placeholder="$t('instance.select')"
    :render-label="renderLabel"
    :filter="filterByName"
    :filterable="true"
    :virtual-scroll="true"
    :fallback-option="false"
    class="bb-instance-select"
    style="width: 12rem"
    @update:value="$emit('update:instance', $event)"
  />
</template>

<script lang="ts" setup>
import { computed, watch, h } from "vue";
import { NSelect, SelectOption } from "naive-ui";
import { useI18n } from "vue-i18n";

import { useInstanceList, useInstanceStore } from "@/store";
import {
  InstanceId,
  Instance,
  EngineType,
  unknown,
  EngineTypeList,
  UNKNOWN_ID,
} from "@/types";
import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";

interface InstanceSelectOption extends SelectOption {
  value: InstanceId;
  instance: Instance;
}

const props = withDefaults(
  defineProps<{
    instance: InstanceId | undefined;
    environment?: string;
    allowedEngineTypeList?: readonly EngineType[];
    includeAll?: boolean;
    includeArchived?: boolean;
    autoReset?: boolean;
    filter?: (instance: Instance, index: number) => boolean;
  }>(),
  {
    environment: undefined,
    allowedEngineTypeList: () => EngineTypeList,
    includeAll: false,
    includeArchived: false,
    autoReset: true,
    filter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:instance", value: InstanceId | undefined): void;
}>();

const { t } = useI18n();
const instanceStore = useInstanceStore();
useInstanceList();

const rawInstanceList = computed(() => {
  let list: Instance[] = [];
  if (props.environment && props.environment !== String(UNKNOWN_ID)) {
    list = instanceStore.getInstanceListByEnvironmentId(props.environment, [
      "NORMAL",
      "ARCHIVED",
    ]);
  } else {
    list = instanceStore.getInstanceList(["NORMAL", "ARCHIVED"]);
  }
  // Filter by engine type
  list = list.filter((instance) =>
    props.allowedEngineTypeList.includes(instance.engine)
  );
  return list;
});

const combinedInstanceList = computed(() => {
  let list = rawInstanceList.value.filter((instance) => {
    if (props.includeArchived) return true;
    if (instance.rowStatus === "NORMAL") return true;
    // ARCHIVED
    if (instance.id === props.instance) return true;
    return false;
  });

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (props.instance === UNKNOWN_ID || props.includeAll) {
    const dummyAll = unknown("INSTANCE");
    dummyAll.name = t("instance.all");
    list.unshift(dummyAll);
  }

  return list;
});

const renderLabel = (option: SelectOption) => {
  const { instance } = option as InstanceSelectOption;
  if (instance.id === UNKNOWN_ID) {
    return t("instance.all");
  }
  const icon = h(InstanceEngineIcon, {
    instance,
    class: "bb-instance-select--engine-icon shrink-0",
  });
  const text = h("span", {}, instance.name);
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
      value: instance.id,
      label: instance.name,
    };
  });
});

const filterByName = (pattern: string, option: SelectOption) => {
  const { instance } = option as InstanceSelectOption;
  return instance.name.toLowerCase().includes(pattern.toLowerCase());
};

// The instance list might change if environment changes, and the previous selected id
// might not exist in the new list. In such case, we need to reset the selection
// and emit the event.
const resetInvalidSelection = () => {
  if (!props.autoReset) return;
  if (
    props.instance &&
    !combinedInstanceList.value.find((item) => item.id === props.instance)
  ) {
    emit("update:instance", undefined);
  }
};

watch([() => props.instance, combinedInstanceList], resetInvalidSelection, {
  immediate: true,
});
</script>

<style>
.bb-instance-select .n-base-selection--active .bb-instance-select--engine-icon {
  opacity: 0.3;
}
</style>
