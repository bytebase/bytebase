<template>
  <template v-for="label in PRESET_LABEL_KEYS" :key="label.key">
    <dd class="flex items-center text-sm md:mr-4">
      <slot name="label" :label="label"></slot>
      <DatabaseLabelPropItem
        :label-key="label"
        :value="getLabelValue(label)"
        :database="database"
        :allow-edit="allowEdit"
        @update:value="(value) => onUpdateValue(label, value)"
      />
    </dd>
  </template>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { withDefaults, watch, reactive } from "vue";
import type { ComposedDatabase } from "@/types";
import { PRESET_LABEL_KEYS } from "@/utils";
import DatabaseLabelPropItem from "./DatabaseLabelPropItem.vue";

const props = withDefaults(
  defineProps<{
    labels: Record<string, string>;
    database: ComposedDatabase;
    allowEdit?: boolean;
  }>(),
  {
    allowEdit: false,
  }
);

const emit = defineEmits<{
  (event: "update:labels", labels: Record<string, string>): void;
}>();

const state = reactive({
  labels: cloneDeep(props.labels),
});

watch(
  () => props.labels,
  (list) => (state.labels = cloneDeep(list))
);

const getLabelValue = (key: string): string | undefined => {
  return state.labels[key] ?? "";
};

const setLabelValue = (key: string, value: string) => {
  if (value) {
    state.labels[key] = value;
  } else {
    delete state.labels[key];
  }
};

const onUpdateValue = (key: string, value: string) => {
  setLabelValue(key, value);
  emit("update:labels", state.labels);
};
</script>
