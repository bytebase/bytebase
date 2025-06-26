<template>
  <div class="flex items-center gap-x-2">
    <!-- Primary action button -->
    <UnifiedActionButton
      v-if="primaryAction"
      :action="primaryAction.action"
      :disabled="primaryAction.disabled"
      @perform-action="handleAction"
    />

    <!-- Dropdown for secondary actions -->
    <NDropdown
      v-if="secondaryActions.length > 0"
      trigger="click"
      placement="bottom-end"
      :options="dropdownOptions"
      :render-option="renderDropdownOption"
      @select="handleDropdownSelect"
    >
      <NButton :quaternary="true" size="medium" style="--n-padding: 0 4px">
        <EllipsisVerticalIcon class="w-5 h-5" />
      </NButton>
    </NDropdown>
  </div>
</template>

<script setup lang="ts">
import { EllipsisVerticalIcon } from "lucide-vue-next";
import { NButton, NDropdown, type DropdownOption } from "naive-ui";
import { computed, h } from "vue";
import type { VNode } from "vue";
import { useI18n } from "vue-i18n";
import { DropdownItemWithErrorList } from "@/components/IssueV1/components/common";
import UnifiedActionButton from "./UnifiedActionButton.vue";
import type { ActionConfig, UnifiedAction } from "./types";

const props = defineProps<{
  primaryAction?: ActionConfig;
  secondaryActions: ActionConfig[];
}>();

const emit = defineEmits<{
  (event: "perform-action", action: UnifiedAction): void;
}>();

const { t } = useI18n();

const actionDisplayName = (action: UnifiedAction): string => {
  switch (action) {
    case "APPROVE":
      return t("common.approve");
    case "REJECT":
      return t("custom-approval.issue-review.send-back");
    case "RE_REQUEST":
      return t("custom-approval.issue-review.re-request-review");
    case "CLOSE":
      return t("issue.batch-transition.close");
    case "REOPEN":
      return t("issue.batch-transition.reopen");
    case "CREATE_ROLLOUT":
      return t("common.create") + " " + t("common.rollout");
    case "CREATE_ISSUE":
      return t("common.create") + " " + t("common.issue");
  }
};

// Removed unused interface - using inline types instead

const dropdownOptions = computed(() => {
  return props.secondaryActions.map((config) => ({
    key: config.action,
    label: actionDisplayName(config.action),
    action: config.action,
    disabled: config.disabled,
    description: config.description,
  }));
});

const renderDropdownOption = ({
  node,
  option,
}: {
  node: VNode;
  option: DropdownOption;
}) => {
  const actionOption = props.secondaryActions.find(
    (config) => config.action === (option as any).key
  );
  const disabled = actionOption?.disabled;
  const description = actionOption?.description;
  const errors = disabled
    ? [
        description ||
          t("issue.error.you-are-not-allowed-to-perform-this-action"),
      ]
    : [];
  return h(
    DropdownItemWithErrorList,
    {
      errors,
      placement: "left",
    },
    {
      default: () => node,
    }
  );
};

const handleAction = (action: UnifiedAction) => {
  emit("perform-action", action);
};

const handleDropdownSelect = (key: string) => {
  const option = dropdownOptions.value.find((opt) => opt.key === key);
  if (option && !option.disabled) {
    handleAction(option.action);
  }
};
</script>
