<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="databaseGroupList"
    :row-clickable="true"
    row-key="name"
    class="border"
  >
    <template #item="{ item }: { item: DatabaseGroup }">
      <div class="bb-grid-cell">
        {{ item.databasePlaceholder }}
      </div>
      <div class="bb-grid-cell">environment name</div>
      <div class="bb-grid-cell gap-x-2">
        <NButton size="small">Configure</NButton>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { BBGridColumn } from "@/bbkit";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

withDefaults(
  defineProps<{
    databaseGroupList: DatabaseGroup[];
  }>(),
  {}
);

const { t } = useI18n();

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: t("common.name"), width: "1fr" },
    {
      title: t("common.environment"),
      width: "1fr",
    },
    {
      title: "",
      width: "10rem",
    },
  ];

  return columns;
});
</script>
