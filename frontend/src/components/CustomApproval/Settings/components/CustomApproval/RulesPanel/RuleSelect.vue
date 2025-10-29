<template>
  <div
    class="flex items-center"
    :class="attrs.class as VueClass"
    :style="attrs.style as VueStyle"
  >
    <label v-if="label" class="mr-2">{{ label }}</label>
    <SpinnerSelect
      :value="value"
      :on-update="onUpdate"
      :options="options"
      :placeholder="$t('custom-approval.approval-flow.select')"
      :consistent-menu-width="false"
      :disabled="disabled || !allowAdmin"
      :filterable="true"
      class="bb-rule-select"
      v-bind="selectAttrs"
    />
    <NButton
      v-if="link"
      quaternary
      type="info"
      class="!rounded !w-[var(--n-height)] !p-0 !ml-1"
      :disabled="!selectedRule || !allowAdmin"
      @click="toApprovalFlow"
    >
      <heroicons:pencil-square class="w-5 h-5" />
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { omit } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { NButton, type SelectProps } from "naive-ui";
import { computed, useAttrs } from "vue";
import { useI18n } from "vue-i18n";
import { SpinnerSelect } from "@/components/v2/Form";
import { useWorkspaceApprovalSettingStore } from "@/store";
import { BUILTIN_APPROVAL_FLOWS, isBuiltinFlowId } from "@/types";
import type { VueClass, VueStyle } from "@/utils";
import { useCustomApprovalContext } from "../context";

export interface ApprovalTemplateSelectorProps
  extends /* @vue-ignore */ SelectProps {
  label?: string;
  link?: boolean;
  disabled?: boolean;
  value?: string;
  onUpdate: (value: string | undefined) => Promise<any>;
  selectClass?: VueClass;
  selectStyle?: VueStyle;
}
const props = defineProps<ApprovalTemplateSelectorProps>();

defineEmits<{
  (event: "update:value", value: string | undefined): void;
}>();

const { t } = useI18n();
const store = useWorkspaceApprovalSettingStore();
const context = useCustomApprovalContext();
const { allowAdmin, hasFeature, showFeatureModal } = context;

const attrs = useAttrs();
const selectAttrs = computed(() => ({
  ...omit(attrs, "class", "style"),
  class: props.selectClass,
  style: props.selectStyle,
}));

const options = computed(() => {
  // Custom flow options (filter out built-in flows that are in the database)
  const customRuleOptions = store.config.rules
    .filter((rule) => !isBuiltinFlowId(rule.template.id))
    .map<SelectOption>((rule) => ({
      label: rule.template.title,
      value: rule.template.id,
    }));

  // Built-in flow options - only show most commonly used ones
  const commonBuiltinFlows = [
    "bb.project-owner",
    "bb.workspace-dba",
    "bb.project-owner-workspace-dba",
    "bb.project-owner-workspace-dba-workspace-admin",
  ];

  const builtinOptions = BUILTIN_APPROVAL_FLOWS.filter((flow) =>
    commonBuiltinFlows.includes(flow.id)
  ).map<SelectOption>((flow) => ({
    label: flow.title,
    value: flow.id,
  }));

  const options: SelectOption[] = [
    { value: "", label: t("custom-approval.approval-flow.skip") },
  ];

  // Add custom flows group FIRST (most likely to be selected)
  if (customRuleOptions.length > 0) {
    options.push({
      type: "group",
      label: t("custom-approval.approval-flow.custom"),
      key: "custom-group",
      children: customRuleOptions,
    } as SelectOption);
  }

  // Add built-in flows group SECOND (less commonly used)
  if (builtinOptions.length > 0) {
    options.push({
      type: "group",
      label: t("custom-approval.approval-flow.built-in"),
      key: "builtin-group",
      children: builtinOptions,
    } as SelectOption);
  }

  return options;
});

const selectedRule = computed(() => {
  return store.config.rules.find((rule) => rule.template.id === props.value);
});

const toApprovalFlow = () => {
  const rule = selectedRule.value;
  if (!rule) {
    return;
  }
  if (!hasFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  context.dialog.value = {
    mode: "EDIT",
    rule,
  };
};
</script>
