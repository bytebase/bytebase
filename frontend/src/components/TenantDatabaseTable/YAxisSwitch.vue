<template>
  <div
    class="pl-1 py-1 relative rounded inline-flex items-center hover:bg-control-bg-hover cursor-pointer select-none"
    @click="switchValue"
  >
    <span>{{ label }}</span>
    <heroicons-solid:selector class="h-4 w-4 text-control-light" />
    <select
      v-if="labelList.length > 2"
      :value="label"
      class="absolute w-full h-full inset-0 opacity-0"
      @change="(e: any) => $emit('update:label', e.target.value)"
    >
      <option v-for="opt in labelList" :key="opt.key" :value="opt.key">
        {{ opt.key }}
      </option>
    </select>
  </div>
</template>

<script lang="ts" setup>
import { defineProps, defineEmits } from "vue";
import { Label, LabelKeyType } from "../../types";

const props = defineProps<{
  label: LabelKeyType;
  labelList: Label[];
}>();

const emit = defineEmits<{
  (event: "update:label", label: LabelKeyType): void;
}>();

const switchValue = () => {
  const list = props.labelList;
  if (list.length === 0) return;
  if (list.length === 1) return;
  if (list.length > 2) return;

  const index = list.findIndex((label) => label.key === props.label);
  if (index < 0) return;
  const next = (index + 1) % list.length;
  emit("update:label", list[next].key);
};
</script>
