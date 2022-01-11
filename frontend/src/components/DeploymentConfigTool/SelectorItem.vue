<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <div class="selector-item" :class="{ editable }">
    <LabelSelect
      v-model:value="selector.key"
      :options="keys"
      :disabled="!editable"
      class="select key"
    />
    <LabelSelect
      v-model:value="selector.operator"
      :options="OPERATORS"
      :disabled="!editable"
      class="select operator"
    />
    <LabelSelect
      v-if="selector.operator === 'In'"
      v-model:value="selector.values"
      :options="values"
      :disabled="!editable"
      :multiple="true"
      :placeholder="$t('label.placeholder.select-values')"
      class="select values"
    />
    <div v-if="editable" class="remove" @click="$emit('remove')">
      <heroicons-outline:x class="w-4 h-4 text-control" />
    </div>
  </div>
</template>

<script lang="ts">
/* eslint-disable vue/no-mutating-props */

import { computed, defineComponent, PropType, watch } from "vue";
import {
  AvailableLabel,
  LabelSelectorRequirement,
  OperatorType,
} from "../../types";
import LabelSelect from "./LabelSelect.vue";

const OPERATORS: OperatorType[] = ["In", "Exists"];

export default defineComponent({
  name: "SelectorItem",
  components: { LabelSelect },
  props: {
    selector: {
      type: Object as PropType<LabelSelectorRequirement>,
      required: true,
    },
    labelList: {
      type: Array as PropType<AvailableLabel[]>,
      default: () => [],
    },
    editable: {
      type: Boolean,
      default: false,
    },
  },
  emits: ["remove"],
  setup(props) {
    const keys = computed(() => props.labelList.map((label) => label.key));
    const values = computed(() => {
      if (!props.selector.key) return [];
      const labelDefinition = props.labelList.find(
        (l) => l.key === props.selector.key
      );
      if (!labelDefinition) return [];
      return labelDefinition.valueList;
    });

    const resetValues = () => {
      props.selector.values = [];
    };

    watch(() => props.selector.key, resetValues);
    watch(() => props.selector.operator, resetValues);

    return { OPERATORS, keys, values };
  },
});
</script>

<style scoped lang="postcss">
.selector-item {
  @apply relative max-w-full flex shadow-sm rounded-md overflow-hidden;
}
.selector-item > * {
  @apply text-sm select-none text-main bg-white border border-control-border rounded-md cursor-default relative z-0;
}
.selector-item.editable > *:hover {
  @apply z-10 bg-control-bg-hover;
}
.selector-item > :not(:first-child, :last-child) {
  @apply rounded-l-none rounded-r-none;
}
.selector-item > :first-child {
  @apply rounded-r-none;
}
.selector-item > :last-child {
  @apply rounded-l-none;
}
.selector-item > :not(:first-child) {
  @apply -ml-px;
}
.select {
  @apply flex items-center relative;
}
.remove {
  @apply flex items-center pl-2 pr-2;
}
</style>
