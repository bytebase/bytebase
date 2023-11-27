<template>
  <TabFilter
    :value="state.selectedTab"
    :items="tabItemList"
    @update:value="
    (val) => {
        state.selectedTab = val as string;
        $emit('update:value', val == 'all' ? undefined : state.selectedTab);
      }
    "
  />
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { CategoryType } from "@/types/sqlReview";

export interface CategoryFilterItem {
  id: CategoryType;
  name: string;
}

interface LocalState {
  selectedTab: string;
}

const props = withDefaults(
  defineProps<{
    value?: string;
    categoryList: CategoryFilterItem[];
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
    },
  ];
  list.push(
    ...props.categoryList.map((category) => {
      return {
        value: category.id,
        label: category.name,
      };
    })
  );
  return list;
});
</script>
