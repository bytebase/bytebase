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
        <slot :rule-list="data.ruleList" />
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
import { convertToCategoryList } from "@/types";

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
  const list = [
    {
      value: "all",
      label: t("common.all"),
      ruleList: props.ruleList,
    },
  ];
  list.push(
    ...convertToCategoryList(props.ruleList).map((category) => {
      return {
        value: category.id,
        label: t(`sql-review.category.${category.id.toLowerCase()}`),
        ruleList: category.ruleList,
      };
    })
  );
  return list;
});
</script>
