<template>
  <div v-if="label" class="flex items-center space-x-2">
    <slot name="title">
      <span class="textlabel mr-1">Group by</span>
    </slot>

    <select class="btn-select py-[0.3rem]" :value="label" @change="onChange">
      <option
        v-for="lbl in labelList"
        :key="lbl.key"
        :value="lbl.key"
        class="capitalize"
      >
        {{ capitalize(hidePrefix(lbl.key)) }}
      </option>
    </select>
  </div>
</template>

<script lang="ts" setup>
import { capitalize } from "lodash-es";
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
  // const target = e.target as HTMLInputElement;
  // if (target.checked) {
  //   emit("update:label", target.value);
  // }
  const target = e.target as HTMLSelectElement;
  emit("update:label", target.value);
};
</script>
