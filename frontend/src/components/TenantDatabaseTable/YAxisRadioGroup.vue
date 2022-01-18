<template>
  <div v-if="label" class="flex items-center space-x-2">
    <slot name="title">
      <label for="group-by">Group by</label>
    </slot>

    <label v-for="lbl in labelList" :key="lbl.key">
      <input
        type="radio"
        :value="lbl.key"
        :checked="lbl.key === label"
        @change="onChange"
      />
      <span class="capitalize ml-1">
        {{ hidePrefix(lbl.key) }}
      </span>
    </label>
  </div>
</template>

<script lang="ts" setup>
import { defineProps, defineEmits } from "vue";
import { Label, LabelKeyType } from "../../types";
import { hidePrefix } from "../../utils";

defineProps<{
  label: LabelKeyType;
  labelList: Label[];
}>();

const emit = defineEmits<{
  (event: "update:label", label: LabelKeyType): void;
}>();

const onChange = (e: Event) => {
  const target = e.target as HTMLInputElement;
  if (target.checked) {
    emit("update:label", target.value);
  }
};
</script>
