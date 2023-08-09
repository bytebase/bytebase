<template>
  <BBTabFilter
    :tab-item-list="tabItemList"
    :selected-index="index"
    @select-index="index = $event"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBTabFilter, type BBTabFilterItem } from "@/bbkit";
import { SupportedSourceList } from "@/types";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { minmax } from "@/utils";
import { sourceText } from "../../common";
import { useRiskFilter } from "./context";

export interface RiskSourceFilterItem {
  value: Risk_Source;
  label: string;
}

const { t } = useI18n();
const { source } = useRiskFilter();

const filterItemList = computed(() => {
  const items: RiskSourceFilterItem[] = [
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

const index = computed({
  get() {
    const index = filterItemList.value.findIndex(
      (item) => item.value === source.value
    );
    if (index < 0) return 0;
    return index;
  },
  set(index) {
    index = minmax(index, 0, filterItemList.value.length - 1);
    source.value = filterItemList.value[index].value;
  },
});

const tabItemList = computed(() => {
  return filterItemList.value.map<BBTabFilterItem>((item) => ({
    title: item.label,
    alert: false,
  }));
});
</script>
