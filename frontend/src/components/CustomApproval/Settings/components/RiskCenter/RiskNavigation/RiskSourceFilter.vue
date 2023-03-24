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
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { useRiskCenterContext } from "../context";
import { SupportedSourceList } from "@/types";
import { minmax } from "@/utils";
import { sourceText } from "../common";

export interface RiskSourceFilterItem {
  value: Risk_Source;
  label: string;
}

const context = useRiskCenterContext();
const { navigation } = context;

const { t } = useI18n();

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
      (item) => item.value === navigation.value.source
    );
    if (index < 0) return 0;
    return index;
  },
  set(index) {
    index = minmax(index, 0, filterItemList.value.length - 1);
    navigation.value.source = filterItemList.value[index].value;
  },
});

const tabItemList = computed(() => {
  return filterItemList.value.map<BBTabFilterItem>((item) => ({
    title: item.label,
    alert: false,
  }));
});
</script>
