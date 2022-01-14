<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <div class="database-label" :class="{ editable }">
    <div class="select key">
      <span>{{ hidePrefix(label.key) }}</span>
      <span v-if="editable" class="dropdown-icon">
        <heroicons-solid:selector class="h-4 w-4 text-control-light" />
      </span>
      <select v-if="editable" v-model="label.key">
        <option v-for="(key, i) in keys" :key="i" :value="key">
          {{ hidePrefix(key) }}
        </option>
      </select>
    </div>
    <div v-if="!editable" class="colon">:</div>
    <div class="select value">
      <span>{{ label.value }}</span>
      <span v-if="editable" class="dropdown-icon">
        <heroicons-solid:selector class="h-4 w-4 text-control-light" />
      </span>
      <select v-if="editable" v-model="label.value">
        <option v-for="(value, i) in values" :key="i" :value="value">
          {{ value }}
        </option>
      </select>
    </div>
    <div v-if="editable" class="remove" @click="$emit('remove')">
      <heroicons-solid:x class="w-3 h-3 text-control" />
    </div>
  </div>
</template>

<script lang="ts">
/* eslint-disable vue/no-mutating-props */

import { computed, defineComponent, PropType, watch } from "vue";
import { DatabaseLabel, Label } from "../../types";
import { hidePrefix } from "../../utils";

export default defineComponent({
  name: "DatabaseLabel",
  props: {
    // bound value will be mutated directly
    // this is an anti-pattern in vue
    // but we do it here to make this component simple and stupid
    label: {
      type: Object as PropType<DatabaseLabel>,
      required: true,
    },
    editable: {
      type: Boolean,
      default: false,
    },
    availableLabels: {
      type: Array as PropType<Label[]>,
      default: () => [],
    },
  },
  emits: ["remove"],
  setup(props) {
    const keys = computed(() =>
      props.availableLabels.map((label) => label.key)
    );
    const values = computed(() => {
      if (!props.label.key) return [];
      const labelDefinition = props.availableLabels.find(
        (l) => l.key === props.label.key
      );
      if (!labelDefinition) return [];
      return labelDefinition.valueList;
    });

    watch(
      () => props.label.key,
      () => {
        if (!props.editable) return false; // if not editable, do nothing

        // otherwise when key changed, reset value selection
        props.label.value = values.value[0] || "";
      }
    );

    return {
      keys,
      values,
      hidePrefix,
    };
  },
});
</script>

<style scoped lang="postcss">
.database-label {
  @apply h-6 relative z-0 flex items-center text-sm rounded overflow-hidden px-2 whitespace-nowrap
   bg-blue-100 border-blue-300 border select-none;
}
.select {
  @apply h-full relative flex items-center overflow-hidden;
}
.select .dropdown-icon {
  @apply ml-0 mr-1 flex items-center p-0 rounded-sm pointer-events-none; /* cursor-pointer hover:bg-gray-300;*/
}
.select select {
  @apply absolute w-full h-full inset-0 p-0 m-0 opacity-0 !important;
}
.colon {
  @apply ml-0.5 mr-2;
}
.remove {
  @apply ml-0 -mr-1 p-px cursor-pointer hover:bg-blue-300 rounded-sm;
}
</style>
