<template>
  <BBTabFilter
    :tab-item-list="tabItemList"
    :selected-index="state.selectedIndex"
    @select-index="
      (index) => {
        state.selectedIndex = index;
        $emit('select', index == 0 ? null : categoryList[index - 1].id);
      }
    "
  />
</template>

<script lang="ts" setup>
import { computed, reactive, PropType, watch } from "vue";
import { BBTabFilterItem } from "@/bbkit/types";
import { useI18n } from "vue-i18n";
import { CategoryType } from "@/types/schemaSystem";

export interface CategoryFilterItem {
  id: CategoryType;
  name: string;
}

interface LocalState {
  selectedIndex: number;
}

const props = defineProps({
  selected: {
    required: false,
    default: undefined,
    type: String,
  },
  categoryList: {
    required: true,
    type: Object as PropType<CategoryFilterItem[]>,
  },
});

const emit = defineEmits(["select"]);

const { t } = useI18n();

const getSelectedIndex = (): number => {
  return props.selected
    ? props.categoryList.findIndex((c) => c.id === props.selected) + 1
    : 0;
};
const state = reactive<LocalState>({
  selectedIndex: getSelectedIndex(),
});

watch(
  () => props.selected,
  () => {
    state.selectedIndex = getSelectedIndex();
  }
);

const tabItemList = computed((): BBTabFilterItem[] => {
  const list: BBTabFilterItem[] = [
    {
      title: t("common.all"),
      alert: false,
    },
  ];
  list.push(
    ...props.categoryList.map((category) => {
      return {
        title: category.name,
        alert: false,
      };
    })
  );
  return list;
});
</script>
