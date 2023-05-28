<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="formatedDatabaseGroupList"
    :row-clickable="true"
    row-key="name"
    class="border"
  >
    <template #item="{ item }: { item: FormatedDatabaseGroup }">
      <div class="bb-grid-cell">
        {{ item.databasePlaceholder }}
      </div>
      <div class="bb-grid-cell">{{ item.environment }}</div>
      <div class="bb-grid-cell gap-x-2">
        <NButton size="small">Configure</NButton>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { BBGridColumn } from "@/bbkit";
import { useEnvironmentV1Store } from "@/store";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { convertDatabaseGroupExprFromCEL } from "@/utils/databaseGroup/cel";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

interface FormatedDatabaseGroup {
  name: string;
  databasePlaceholder: string;
  environment: string;
}

const props = defineProps<{
  databaseGroupList: DatabaseGroup[];
}>();

const { t } = useI18n();
const environmentStore = useEnvironmentV1Store();
const formatedDatabaseGroupList = ref<FormatedDatabaseGroup[]>([]);

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

watch(
  () => [props.databaseGroupList],
  async () => {
    const list: FormatedDatabaseGroup[] = [];
    for (const databaseGroup of props.databaseGroupList) {
      const result = await convertDatabaseGroupExprFromCEL(
        databaseGroup.databaseExpr?.expression ?? ""
      );
      const environment = environmentStore.getEnvironmentByName(
        result.environmentId
      );
      list.push({
        name: databaseGroup.name,
        databasePlaceholder: databaseGroup.databasePlaceholder,
        environment: environment?.title || "",
      });
    }
    formatedDatabaseGroupList.value = list;
  },
  {
    immediate: true,
  }
);
</script>
