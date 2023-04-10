<template>
  <NSelect
    :value="instance"
    :options="options"
    :placeholder="$t('instance.select')"
    style="width: 12rem"
    @update:value="$emit('update:instance', $event)"
  />
</template>

<script lang="ts" setup>
import { computed, watch, h } from "vue";
import { NSelect, SelectOption } from "naive-ui";
import { useI18n } from "vue-i18n";

import { useInstanceStore } from "@/store";
import {
  InstanceId,
  EnvironmentId,
  Instance,
  UNKNOWN_ID,
  unknown,
  EngineType,
  EngineTypeList,
} from "@/types";
import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";

const props = withDefaults(
  defineProps<{
    instance: InstanceId | undefined;
    environment?: EnvironmentId;
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

const rawInstanceList = computed(() => {
  let list: Instance[] = [];
  if (props.environment && props.environment !== UNKNOWN_ID) {
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

const options = computed(() => {
  return combinedInstanceList.value.map<SelectOption>((instance) => {
    return {
      value: instance.id,
      label: () => {
        if (instance.id === UNKNOWN_ID) {
          return t("instance.all");
        }
        const icon = h(InstanceEngineIcon, { instance, class: "shrink-0" });
        const text = h("span", {}, instance.name);
        return h(
          "div",
          {
            class: "flex items-center gap-x-2",
          },
          [icon, text]
        );
      },
    };
  });
});

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
