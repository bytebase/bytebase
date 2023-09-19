<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="databaseGroupList"
    :row-clickable="true"
    row-key="name"
    class="border"
    @click-row="clickDatabaseGroup"
  >
    <template #item="{ item }: { item: ComposedDatabaseGroup }">
      <div class="bb-grid-cell">
        {{ item.databasePlaceholder }}
      </div>
      <div class="bb-grid-cell">{{ item.environment.title }}</div>
      <div v-if="props.showEdit" class="bb-grid-cell gap-x-2 justify-end">
        <NButton size="small" @click.stop="$emit('edit', item)">{{
          $t("common.configure")
        }}</NButton>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBGridColumn } from "@/bbkit";
import { ComposedDatabaseGroup } from "@/types";
import { DatabaseGroup } from "@/types/proto/v1/project_service";

const props = defineProps<{
  databaseGroupList: ComposedDatabaseGroup[];
  showEdit?: boolean;
}>();

defineEmits<{
  (event: "edit", databaseGroup: DatabaseGroup): void;
}>();

const { t } = useI18n();
const router = useRouter();

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: t("common.name"), width: "1fr" },
    {
      title: t("common.environment"),
      width: "1fr",
    },
  ];

  if (props.showEdit) {
    columns.push({
      title: "",
      width: "10rem",
    });
  }

  return columns;
});

const clickDatabaseGroup = (databaseGroup: ComposedDatabaseGroup) => {
  if (!props.showEdit) {
    return;
  }

  router.push(
    `/${databaseGroup.project.name}/database-groups/${databaseGroup.databaseGroupName}`
  );
};
</script>
