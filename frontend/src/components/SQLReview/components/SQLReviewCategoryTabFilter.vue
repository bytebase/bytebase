<template>
  <NTabs
    :value="value"
    :size="'small'"
    :type="tabType"
    @update:value="$emit('update:value', $event)"
  >
    <NTabPane v-for="data in tabItemList" :key="data.value" :name="data.value">
      <template #tab>
        {{ data.label }}
      </template>
      <template #default>
        <slot
          :rule-list="data.value === 'all' ? tabItemList.slice(1) : [data]"
        />
      </template>
    </NTabPane>
  </NTabs>
</template>

<script lang="ts" setup>
import { useWindowSize } from "@vueuse/core";
import { NTabPane, NTabs } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { RuleTemplateV2 } from "@/types";
import { convertToCategoryMap } from "@/types";

export interface RuleListWithCategory {
  value: string;
  label: string;
  ruleList: RuleTemplateV2[];
}

const props = defineProps<{
  value: string;
  ruleList: RuleTemplateV2[];
}>();

defineEmits<{
  (event: "update:value", value: string): void;
}>();

const { t } = useI18n();

const { width: winWidth } = useWindowSize();

const tabType = computed(() => {
  if (winWidth.value >= 1000) {
    return "segment";
  }
  return "line";
});

const tabItemList = computed(() => {
  const list: RuleListWithCategory[] = [
    {
      value: "all",
      label: t("common.all"),
      ruleList: [],
    },
  ];

  for (const [category, ruleList] of convertToCategoryMap(
    props.ruleList
  ).entries()) {
    list.push({
      value: category,
      label: t(`sql-review.category.${category.toLowerCase()}`),
      ruleList: ruleList,
    });
  }
  return list;
});
</script>
