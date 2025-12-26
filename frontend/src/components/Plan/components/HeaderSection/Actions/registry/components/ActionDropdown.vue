<template>
  <NDropdown
    trigger="click"
    placement="bottom-end"
    :options="dropdownOptions"
    :disabled="globalDisabled"
    @select="handleSelect"
  >
    <NButton
      class="px-1!"
      quaternary
      size="medium"
      :disabled="globalDisabled"
      :title="globalDisabled ? globalDisabledReason : undefined"
    >
      <EllipsisVerticalIcon class="w-4 h-4" />
    </NButton>
  </NDropdown>
</template>

<script setup lang="ts">
import { EllipsisVerticalIcon } from "lucide-vue-next";
import { type DropdownOption, NButton, NDropdown } from "naive-ui";
import { computed } from "vue";
import type { ActionContext, ActionDefinition, UnifiedAction } from "../types";

const props = defineProps<{
  actions: ActionDefinition[];
  context: ActionContext;
  globalDisabled?: boolean;
  globalDisabledReason?: string;
  isActionDisabled: (action: ActionDefinition) => boolean;
  getDisabledReason: (action: ActionDefinition) => string | undefined;
}>();

const emit = defineEmits<{
  (e: "execute", id: UnifiedAction): void;
}>();

const dropdownOptions = computed((): DropdownOption[] => {
  return props.actions.map((action) => {
    const disabled = props.isActionDisabled(action);
    const reason = disabled ? props.getDisabledReason(action) : undefined;
    return {
      key: action.id,
      label: action.label(props.context),
      disabled,
      props: reason ? { title: reason } : undefined,
    };
  });
});

const handleSelect = (key: string) => {
  const action = props.actions.find((a) => a.id === key);
  if (action && !props.isActionDisabled(action)) {
    emit("execute", action.id);
  }
};
</script>
