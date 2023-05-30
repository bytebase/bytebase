<template>
  <BBSelect
    :multiple="true"
    :selected-item-list="selectedTypeList"
    :item-list="typeList"
    :placeholder="$t('audit-log.table.select-type')"
    @update-item-list="(list: AuditActivityType[]) => $emit('update-selected-type-list', list)"
  >
    <template #menuItem="{ item }">
      {{ $t(getLabel(item)) }}
    </template>
    <template #menuItemGroup="{ itemList }">
      <ResponsiveTags
        :tags="itemList.map((item: AuditActivityType) => $t(getLabel(item)))"
      />
    </template>
  </BBSelect>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { AuditActivityType, AuditActivityTypeI18nNameMap } from "@/types";

defineProps({
  selectedTypeList: {
    type: Array as PropType<AuditActivityType[]>,
    required: true,
  },
});
defineEmits(["update-selected-type-list"]);

const typeList = Object.keys(AuditActivityTypeI18nNameMap);
const getLabel = (type: AuditActivityType): string =>
  AuditActivityTypeI18nNameMap[type];
</script>
