<template>
  <NDropdown
    :show="showDropdown"
    :options="dropdownOptions"
    placement="bottom-start"
    @select="handleSelect"
    @clickoutside="showDropdown = false"
  >
    <NButton
      :type="isActive ? 'primary' : 'default'"
      size="medium"
      @click="showDropdown = !showDropdown"
    >
      {{ buttonLabel }}
      <template #icon>
        <ChevronDownIcon class="w-4 h-4" />
      </template>
    </NButton>
  </NDropdown>
</template>

<script lang="ts" setup>
import { ChevronDownIcon } from "lucide-vue-next";
import { NButton, NDropdown, type DropdownOption } from "naive-ui";
import { computed, h, ref } from "vue";
import type { SearchParams, SearchScopeId } from "@/utils";
import { getValueFromSearchParams } from "@/utils";

const props = defineProps<{
  scopeId: SearchScopeId;
  label: string;
  options: Array<{ value: string; label: string }>;
  params: SearchParams;
  multiple?: boolean;
}>();

const emit = defineEmits<{
  (event: "select", value: string): void;
}>();

const showDropdown = ref(false);

const currentValue = computed(() => {
  return getValueFromSearchParams(props.params, props.scopeId);
});

const isActive = computed(() => {
  return !!currentValue.value;
});

const buttonLabel = computed(() => {
  if (currentValue.value) {
    const option = props.options.find((o) => o.value === currentValue.value);
    return `${props.label}: ${option?.label || currentValue.value}`;
  }
  return props.label;
});

const dropdownOptions = computed((): DropdownOption[] => {
  return props.options.map((option) => ({
    key: option.value,
    label: option.label,
    type: "render" as const,
    render: () =>
      h(
        "div",
        {
          class: "px-3 py-2 text-sm hover:bg-gray-100 cursor-pointer flex items-center gap-x-2",
        },
        [
          h("input", {
            type: props.multiple ? "checkbox" : "radio",
            checked: option.value === currentValue.value,
            class: "pointer-events-none",
            onChange: (e: Event) => e.preventDefault(),
          }),
          h("span", option.label),
        ]
      ),
  }));
});

const handleSelect = (key: string) => {
  emit("select", key);
  if (!props.multiple) {
    showDropdown.value = false;
  }
};
</script>
