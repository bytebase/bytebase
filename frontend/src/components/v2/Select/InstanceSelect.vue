<template>
  <NSelect
    :value="instance"
    :options="options"
    :placeholder="$t('instance.select')"
    :render-label="renderLabel"
    :filter="filterByTitle"
    :filterable="true"
    :virtual-scroll="true"
    :fallback-option="false"
    class="bb-instance-select"
    style="width: 12rem"
    @update:value="$emit('update:instance', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect, SelectOption } from "naive-ui";
import { computed, watch, h } from "vue";
import { useI18n } from "vue-i18n";
import { useEnvironmentV1Store, useInstanceV1List } from "@/store";
import { ComposedInstance, UNKNOWN_ID, unknownInstance } from "@/types";
import { Engine, State } from "@/types/proto/v1/common";
import { supportedEngineV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../Model/Instance";

interface InstanceSelectOption extends SelectOption {
  value: string;
  instance: ComposedInstance;
}

const props = withDefaults(
  defineProps<{
    instance: string | undefined;
    environment?: string;
    allowedEngineList?: readonly Engine[];
    includeAll?: boolean;
    includeArchived?: boolean;
    autoReset?: boolean;
    filter?: (instance: ComposedInstance, index: number) => boolean;
  }>(),
  {
    environment: undefined,
    allowedEngineList: () => supportedEngineV1List(),
    includeAll: false,
    includeArchived: false,
    autoReset: true,
    filter: undefined,
  }
);

const emit = defineEmits<{
  (event: "update:instance", value: string | undefined): void;
}>();

const { t } = useI18n();
const { instanceList: allInstanceList, ready } = useInstanceV1List(
  true /* showDeleted */
);

const rawInstanceList = computed(() => {
  let list = [...allInstanceList.value];
  if (props.environment && props.environment !== String(UNKNOWN_ID)) {
    const environment = useEnvironmentV1Store().getEnvironmentByUID(
      props.environment
    );
    list = allInstanceList.value.filter(
      (instance) => instance.environment === environment.name
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
    if (instance.uid === props.instance) return true;
    return false;
  });

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (props.instance === String(UNKNOWN_ID) || props.includeAll) {
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
  if (instance.uid === String(UNKNOWN_ID)) {
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
      value: instance.uid,
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
    props.instance &&
    !combinedInstanceList.value.find((item) => item.uid === props.instance)
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
