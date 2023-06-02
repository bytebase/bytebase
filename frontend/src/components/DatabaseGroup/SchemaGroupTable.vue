<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="formatedSchemaGroupList"
    :row-clickable="true"
    row-key="name"
    class="border"
    @click-row="clickSchemaGroup"
  >
    <template #item="{ item }: { item: FormatedSchemaGroup }">
      <div class="bb-grid-cell">
        {{ item.resourceId }}
      </div>
      <div class="bb-grid-cell gap-x-2 justify-end">
        <NButton size="small" @click.stop="$emit('edit', item.schemaGroup)">{{
          $t("common.configure")
        }}</NButton>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn } from "@/bbkit";
import { SchemaGroup } from "@/types/proto/v1/project_service";
import { useRouter } from "vue-router";
import { getProjectNameAndDatabaseGroupNameAndSchemaGroupName } from "@/store/modules/v1/common";

interface FormatedSchemaGroup {
  resourceId: string;
  schemaGroup: SchemaGroup;
}

const props = defineProps<{
  schemaGroupList: SchemaGroup[];
}>();

defineEmits<{
  (event: "edit", schemaGroup: SchemaGroup): void;
}>();

const { t } = useI18n();
const router = useRouter();
const formatedSchemaGroupList = ref<FormatedSchemaGroup[]>([]);

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

const clickSchemaGroup = ({ schemaGroup }: FormatedSchemaGroup) => {
  const [projectName, databaseGroupName, schemaGroupName] =
    getProjectNameAndDatabaseGroupNameAndSchemaGroupName(schemaGroup.name);
  router.push(
    `/projects/${projectName}/database-groups/${databaseGroupName}/schema-groups/${schemaGroupName}`
  );
};

watch(
  () => [props.schemaGroupList],
  () => {
    const list: FormatedSchemaGroup[] = [];
    for (const schemaGroup of props.schemaGroupList) {
      list.push({
        resourceId: schemaGroup.name.split("/").pop() || "",
        schemaGroup,
      });
    }
    formatedSchemaGroupList.value = list;
  },
  {
    immediate: true,
  }
);
</script>
