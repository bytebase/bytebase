<template>
  <BBGrid
    class="border"
    :column-list="COLUMNS"
    :data-source="riskStore.riskList"
  >
    <template #item="{ item }">
      <div class="bb-grid-cell !pl-2 !pr-1">
        {{ item.name }}
      </div>
      <div class="bb-grid-cell gap-x-1 !pl-1 !pr-2">
        <NButton size="tiny">
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
import { NButton } from "naive-ui";

import { BBGrid, type BBGridColumn } from "@/bbkit";
import { useRiskStore } from "@/store";

const { t } = useI18n();
const riskStore = useRiskStore();

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
</script>
