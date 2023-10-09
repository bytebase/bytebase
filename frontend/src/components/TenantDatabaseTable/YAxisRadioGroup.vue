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
import { computed, withDefaults } from "vue";
import { ComposedDatabase, LabelKeyType } from "@/types";
import { displayLabelKey, getAvailableLabelKeyList } from "@/utils";

const props = withDefaults(
  defineProps<{
    databaseList: ComposedDatabase[];
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
  return getAvailableLabelKeyList(
    props.databaseList,
    true /* withReserved */,
    true /* withPreset */,
    true /* sort */
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
