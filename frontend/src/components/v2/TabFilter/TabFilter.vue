<template>
  <div>
    <div v-if="responsive" class="block sm:hidden">
      <label for="tabs" class="sr-only">Select a tab</label>
      <NSelect
        :disabled="disabled"
        :options="items"
        :value="value"
        :consistent-menu-width="false"
        @update:value="(value: ValueType, option: TabFilterItem<ValueType>) => $emit('update:value', value, option)"
      />
    </div>
    <div
      class="gap-x-1 w-full overflow-x-auto hide-scrollbar whitespace-nowrap py-[3px]"
      aria-label="Tabs"
      :class="responsive ? 'hidden sm:flex' : 'flex'"
    >
      <button
        v-for="item in items"
        :key="item.value"
        :disabled="disabled"
        class="rounded-md text-sm px-3 py-1 flex items-center disabled:cursor-not-allowed transition-colors duration-150"
        :class="[
          value === item.value
            ? 'bg-gray-200 text-gray-800 disabled:bg-gray-100'
            : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100 disabled:text-gray-300 disabled:bg-transparent',
        ]"
        @click.prevent="update(item.value)"
      >
        <slot name="label" :item="item">
          {{ item.label }}
        </slot>
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NSelect } from "naive-ui";
import { TabFilterItem } from "./types";

type ValueType = string | number; // Use generic typed component in the future

const props = withDefaults(
  defineProps<{
    value?: ValueType;
    items: TabFilterItem<ValueType>[];
    disabled?: boolean;
    responsive?: boolean;
  }>(),
  {
    value: undefined,
    disabled: false,
    responsive: true,
  }
);

const emit = defineEmits<{
  (
    event: "update:value",
    value: ValueType,
    item: TabFilterItem<ValueType>
  ): void;
}>();

const update = (value: ValueType) => {
  const item = props.items.find((item) => item.value === value);
  if (!item) return;
  emit("update:value", value, item);
};
</script>
