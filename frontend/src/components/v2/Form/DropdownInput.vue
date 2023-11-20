<template>
  <NDropdown
    v-if="allowInputValue"
    trigger="click"
    placement="bottom-start"
    :options="dropdownOptions"
    :width="$attrs.consistentMenuWidth ? 'trigger' : undefined"
    v-bind="dropdownProps"
    @select="$emit('update:value', $event as string)"
  >
    <NInput
      :value="value"
      v-bind="$attrs"
      @update:value="$emit('update:value', $event)"
    >
      <template #suffix>
        <!-- use the same icon and style with NSelect -->
        <NElement
          tag="button"
          style="color: var(--placeholder-color)"
          class="absolute top-1/2 right-[9px] -translate-y-1/2 cursor-pointer"
          :class="suffixClass"
          :style="suffixStyle"
        >
          <svg
            viewBox="0 0 16 16"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            class="w-4 h-4"
          >
            <path
              d="M3.14645 5.64645C3.34171 5.45118 3.65829 5.45118 3.85355 5.64645L8 9.79289L12.1464 5.64645C12.3417 5.45118 12.6583 5.45118 12.8536 5.64645C13.0488 5.84171 13.0488 6.15829 12.8536 6.35355L8.35355 10.8536C8.15829 11.0488 7.84171 11.0488 7.64645 10.8536L3.14645 6.35355C2.95118 6.15829 2.95118 5.84171 3.14645 5.64645Z"
              fill="currentColor"
            ></path>
          </svg>
        </NElement>
      </template>
    </NInput>
  </NDropdown>
  <NSelect
    v-else
    :value="value"
    :options="options"
    v-bind="$attrs"
    @update:value="$emit('update:value', $event as string)"
  />
</template>

<script setup lang="ts">
import {
  DropdownOption,
  DropdownProps,
  NDropdown,
  NElement,
  NInput,
  NSelect,
  SelectOption,
} from "naive-ui";
import { computed } from "vue";
import { VueClass, VueStyle } from "@/utils";

const props = withDefaults(
  defineProps<{
    value: string | undefined | null;
    allowInputValue?: boolean;
    options: SelectOption[];
    dropdownProps?: DropdownProps;
    suffixClass?: VueClass;
    suffixStyle?: VueStyle;
  }>(),
  {
    allowInputValue: true,
    dropdownProps: undefined,
    suffixClass: undefined,
    suffixStyle: undefined,
  }
);
defineEmits<{
  (event: "update:value", value: string): void;
}>();

const dropdownOptions = computed(() => {
  return props.options.map<DropdownOption>((opt) => ({
    key: opt.value,
    ...opt,
  }));
});
</script>
