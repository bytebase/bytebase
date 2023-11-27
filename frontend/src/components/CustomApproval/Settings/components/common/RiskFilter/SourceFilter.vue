<template>
  <TabFilter v-model:value="source" :items="filterItemList" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { SupportedSourceList } from "@/types";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { sourceText } from "../../common";
import { useRiskFilter } from "./context";

export interface RiskSourceFilterItem {
  value: Risk_Source;
  label: string;
}

const { t } = useI18n();
const { source } = useRiskFilter();

const filterItemList = computed(() => {
  const items = [
    {
      value: Risk_Source.SOURCE_UNSPECIFIED,
      label: t("common.all"),
    },
  ];
  SupportedSourceList.forEach((source) => {
    items.push({
      value: source,
      label: sourceText(source),
    });
  });
  return items;
});
</script>
