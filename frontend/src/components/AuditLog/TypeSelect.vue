<template>
  <BBSelect
    :multiple="true"
    :selected-item-list="selectedTypeList"
    :item-list="typeList"
    :placeholder="$t('audit-log.table.select-type')"
    @update-item-list="(list: LogEntity_Action[]) => $emit('update-selected-type-list', list)"
  >
    <template #menuItem="{ item }">
      {{ $t(getLabel(item)) }}
    </template>
    <template #menuItemGroup="{ itemList }">
      <ResponsiveTags
        :tags="itemList.map((item: LogEntity_Action) => $t(getLabel(item)))"
      />
    </template>
  </BBSelect>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { AuditActivityTypeI18nNameMap } from "@/types";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";

defineProps({
  selectedTypeList: {
    type: Array as PropType<LogEntity_Action[]>,
    required: true,
  },
});
defineEmits(["update-selected-type-list"]);

const typeList = Object.keys(AuditActivityTypeI18nNameMap);
const getLabel = (type: LogEntity_Action): string =>
  AuditActivityTypeI18nNameMap[type];
</script>
