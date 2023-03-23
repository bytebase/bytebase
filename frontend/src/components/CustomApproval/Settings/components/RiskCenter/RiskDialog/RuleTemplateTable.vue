<template>
  <BBGrid
    class="border"
    :column-list="COLUMNS"
    :data-source="filteredTemplateList"
    :row-clickable="false"
    row-key="key"
  >
    <template #item="{ item: tpl }: { item: RuleTemplate }">
      <div class="bb-grid-cell !pl-2 !pr-1">
        {{ titleOfTemplate(tpl) }}
      </div>
      <div class="bb-grid-cell gap-x-1 !pl-1 !pr-2">
        <NButton size="tiny" @click="applyTemplate(tpl)">
          {{ $t("custom-approval.security-rule.template.load") }}
        </NButton>
        <NButton size="tiny">
          {{ $t("common.view") }}
        </NButton>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { NButton, useDialog } from "naive-ui";

import { BBGrid, type BBGridColumn } from "@/bbkit";
import { type RuleTemplate, useRuleTemplates } from "./template";
import { useRiskCenterContext } from "../context";
import { defer } from "@/utils";
import { Risk, Risk_Source } from "@/types/proto/v1/risk_service";
import { buildCELExpr, SimpleExpr } from "@/plugins/cel";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";

const props = defineProps<{
  dirty?: boolean;
}>();

const emit = defineEmits<{
  (event: "apply-template", overrides: Partial<Risk>, expr: SimpleExpr): void;
}>();

const { t, te } = useI18n();
const { dialog } = useRiskCenterContext();
const templateList = useRuleTemplates();
const nDialog = useDialog();

const COLUMNS = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("custom-approval.security-rule.rule-name"),
      width: "1fr",
      class: "!pl-2 !pr-1 !py-0.5",
    },
    {
      title: t("common.operations"),
      width: "auto",
      class: "!px-1",
    },
  ];
  return columns;
});

const filteredTemplateList = computed(() => {
  let list = [...templateList.value];
  const { mode, risk } = dialog.value!;
  if (mode === "EDIT") {
    const { source } = risk;
    list = list.filter((tpl) => {
      return (
        tpl.source === Risk_Source.SOURCE_UNSPECIFIED || tpl.source === source
      );
    });
  }
  return list;
});

const titleOfTemplate = (template: RuleTemplate) => {
  const { key } = template;
  const keypath = `custom-approval.security-rule.template.presets.${key}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return key;
};

const confirmApplyTemplate = async () => {
  if (dialog.value?.mode === "CREATE" && !props.dirty) {
    return true;
  }
  const d = defer<boolean>();
  nDialog.warning({
    title: t("custom-approval.security-rule.template.load-template"),
    content: t("common.will-override-current-data"),
    maskClosable: false,
    closeOnEsc: false,
    positiveText: t("common.confirm"),
    negativeText: t("common.cancel"),
    onPositiveClick: () => d.resolve(true),
    onNegativeClick: () => d.resolve(false),
  });
  return d.promise;
};

const applyTemplate = async (template: RuleTemplate) => {
  if (!(await confirmApplyTemplate())) {
    return;
  }

  const { expr, source, level } = template;
  const title = titleOfTemplate(template);
  const { mode } = dialog.value!;
  const overrides: Partial<Risk> = {
    expression: ParsedExpr.fromJSON(buildCELExpr(expr)),
    level,
    title,
  };
  if (mode === "CREATE" && source !== Risk_Source.SOURCE_UNSPECIFIED) {
    overrides.source = source;
  }
  emit("apply-template", overrides, expr);
};
</script>
