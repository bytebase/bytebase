<template>
  <div>
    <div v-if="responsive" class="block sm:hidden">
      <label for="tabs" class="sr-only">Select a tab</label>
      <select
        id="tabs"
        name="tabs"
        class="control block w-full"
        :disabled="disabled"
        @change="update(($event.target as HTMLSelectElement).value)"
      >
        <option
          v-for="item in items"
          :key="item.value"
          :value="item.value"
          :selected="value === item.value"
        >
          {{ item.label }}
        </option>
      </select>
    </div>
    <div :class="responsive && 'hidden sm:block'">
      <div
        class="flex space-x-4 w-full overflow-x-auto hide-scrollbar"
        aria-label="Tabs"
      >
        <button
          v-for="item in items"
          :key="item.value"
          :disabled="disabled"
          class="tab px-3 py-1 flex items-center whitespace-nowrap disabled:cursor-not-allowed"
          :class="[
            value === item.value
              ? 'bg-gray-200 text-gray-800 disabled:bg-gray-100'
              : 'text-gray-500 hover:text-gray-700',
          ]"
          @click.prevent="update(item.value)"
        >
          {{ item.label }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { TabFilterItem } from "./types";

type ValueType = string | number; // Use generic typed component in the future

const props = withDefaults(
  defineProps<{
    value: ValueType;
    items: TabFilterItem<ValueType>[];
    disabled?: boolean;
    responsive?: boolean;
  }>(),
  {
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
