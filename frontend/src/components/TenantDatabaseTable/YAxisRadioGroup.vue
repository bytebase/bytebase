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
        {{ displayLabelKey(key) }}
      </option>
    </select>
  </div>
</template>

<script lang="ts" setup>
import { orderBy, uniq } from "lodash-es";
import { computed, withDefaults } from "vue";
import { Database } from "@/types/proto/v1/database_service";
import { LabelKeyType } from "../../types";
import {
  displayLabelKey,
  isPresetLabel,
  isReservedLabel,
  PRESET_LABEL_KEYS,
  RESERVED_LABEL_KEYS,
} from "../../utils";

const props = withDefaults(
  defineProps<{
    databaseList: Database[];
    label: LabelKeyType;
    excludedKeyList?: LabelKeyType[];
  }>(),
  {
    excludedKeyList: () => [],
  }
);

const emit = defineEmits<{
  (event: "update:label", label: LabelKeyType): void;
}>();

const labelKeyList = computed(() => {
  const keys = uniq(props.databaseList.flatMap((db) => Object.keys(db.labels)));
  [...RESERVED_LABEL_KEYS, ...PRESET_LABEL_KEYS].forEach((key) => {
    if (!keys.includes(key)) {
      keys.push(key);
    }
  });
  return orderBy(
    keys,
    [
      (key) => (isReservedLabel(key) ? -1 : 1),
      (key) => (isPresetLabel(key) ? -1 : 1),
      (key) => key,
    ],
    ["asc", "asc", "asc"]
  ).filter((key) => !props.excludedKeyList.includes(key));
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
