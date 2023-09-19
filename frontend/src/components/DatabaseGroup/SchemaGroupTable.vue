<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="schemaGroupList"
    :row-clickable="true"
    row-key="name"
    class="border"
    @click-row="clickSchemaGroup"
  >
    <template #item="{ item }: { item: ComposedSchemaGroup }">
      <div class="bb-grid-cell">
        {{ item.tablePlaceholder }}
      </div>
      <div class="bb-grid-cell gap-x-2 justify-end">
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
import { getProjectNameAndDatabaseGroupNameAndSchemaGroupName } from "@/store/modules/v1/common";
import { ComposedSchemaGroup } from "@/types";
import { SchemaGroup } from "@/types/proto/v1/project_service";

defineProps<{
  schemaGroupList: SchemaGroup[];
}>();

defineEmits<{
  (event: "edit", schemaGroup: SchemaGroup): void;
}>();

const { t } = useI18n();
const router = useRouter();

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: t("common.name"), width: "1fr" },
    {
      title: "",
      width: "10rem",
    },
  ];

  return columns;
});

const clickSchemaGroup = (schemaGroup: ComposedSchemaGroup) => {
  const [projectName, databaseGroupName, schemaGroupName] =
    getProjectNameAndDatabaseGroupNameAndSchemaGroupName(schemaGroup.name);
  router.push(
    `/projects/${projectName}/database-groups/${databaseGroupName}/table-groups/${schemaGroupName}`
  );
};
</script>
