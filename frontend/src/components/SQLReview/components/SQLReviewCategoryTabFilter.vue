<template>
  <NTabs
    :value="state.selectedTab"
    :size="'small'"
    :type="tabType"
    @update:value="
      (val: string) => {
        state.selectedTab = val;
        $emit('update:value', val == 'all' ? undefined : state.selectedTab);
      }
    "
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
import { NTabs, NTabPane } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { RuleTemplateV2 } from "@/types";
import { convertToCategoryMap } from "@/types";

export interface RuleListWithCategory {
  value: string;
  label: string;
  ruleList: RuleTemplateV2[];
}

interface LocalState {
  selectedTab: string;
}

const props = withDefaults(
  defineProps<{
    value?: string;
    ruleList: RuleTemplateV2[];
  }>(),
  {
    value: undefined,
  }
);

defineEmits<{
  (event: "update:value", value: string | undefined): void;
}>();

const { t } = useI18n();

const state = reactive<LocalState>({
  selectedTab: "all",
});

const { width: winWidth } = useWindowSize();

const tabType = computed(() => {
  if (winWidth.value >= 1000) {
    return "segment";
  }
  return "line";
});

watch(
  () => props.value,
  (selected) => {
    state.selectedTab = selected ?? "all";
  }
);

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
