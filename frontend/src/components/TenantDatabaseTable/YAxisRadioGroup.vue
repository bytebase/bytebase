<template>
  <div v-if="visible" class="flex items-center space-x-2">
    <slot name="title">
      <span class="textlabel mr-1">Group by</span>
    </slot>

    <select class="btn-select py-[0.5rem]" :value="label" @change="onChange">
      <option
        v-for="key in labelKeyList"
        :key="key"
        :value="key"
        class="capitalize"
      >
        {{ capitalize(hidePrefix(key)) }}
      </option>
    </select>
  </div>
</template>

<script lang="ts" setup>
import { capitalize } from "lodash-es";
import { computed, withDefaults } from "vue";
import { LabelKeyType } from "../../types";
import {
  hidePrefix,
  PRESET_LABEL_KEYS,
  RESERVED_LABEL_KEYS,
} from "../../utils";

const props = withDefaults(
  defineProps<{
    label: LabelKeyType;
    excludedKeyList?: LabelKeyType[];
  }>(),
  {
    excludedKeyList: () => [],
  }
);

const LABEL_KEY_LIST = [...RESERVED_LABEL_KEYS, ...PRESET_LABEL_KEYS];

const emit = defineEmits<{
  (event: "update:label", label: LabelKeyType): void;
}>();

const labelKeyList = computed(() => {
  return LABEL_KEY_LIST.filter((key) => !props.excludedKeyList.includes(key));
});

const visible = computed(() => {
  if (!props.label) return false;
  return labelKeyList.value.includes(props.label);
});

const onChange = (e: Event) => {
  const target = e.target as HTMLSelectElement;
  emit("update:label", target.value);
};
</script>
