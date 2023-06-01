<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="databaseGroupList"
    row-key="name"
    class="border"
    :row-clickable="true"
    @click-row="handleDatabaseGroupSelect"
  >
    <template #item="{ item: dbGroup }: { item: ComposedDatabaseGroup }">
      <div class="bb-grid-cell justify-center items-center">
        <NRadio
          :checked="state.selectedDatabaseGroupName === dbGroup.name"
          :value="dbGroup.name"
          name="database-group"
          @update:checked="handleDatabaseGroupSelect(dbGroup)"
        >
        </NRadio>
      </div>
      <div class="bb-grid-cell">
        {{ dbGroup.databaseGroupName }}
      </div>
      <div class="bb-grid-cell">{{ dbGroup.project.title }}</div>
      <div class="bb-grid-cell">{{ dbGroup.environment.title }}</div>
      <template v-if="state.selectedDatabaseGroupName === dbGroup.name">
        <template
          v-for="schemaGroup in getSchemaGroupListByDBGroupName(dbGroup.name)"
          :key="schemaGroup.name"
        >
          <div class="bb-grid-cell"></div>
          <div class="bb-grid-cell">
            <NCheckbox
              :checked="
                state.selectedSchemaGroupNameList?.includes(schemaGroup.name)
              "
              @update-checked="handleSchemaGroupCheck(schemaGroup)"
            >
              <span>{{ getSchemaGroupTitle(schemaGroup.name) }}</span>
            </NCheckbox>
          </div>
          <div class="bb-grid-cell"></div>
          <div class="bb-grid-cell"></div>
        </template>
      </template>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { NRadio, NCheckbox } from "naive-ui";
import { ref, watch, reactive, computed } from "vue";
import { useDBGroupStore } from "@/store";
import { ComposedDatabaseGroup } from "@/types";
import { SchemaGroup } from "@/types/proto/v1/project_service";
import { getProjectNameAndDatabaseGroupNameAndSchemaGroupName } from "@/store/modules/v1/common";
import { BBGridColumn } from "@/bbkit";
import { useI18n } from "vue-i18n";

interface LocalState {
  selectedDatabaseGroupName?: string;
  selectedSchemaGroupNameList: string[];
}

const props = defineProps<{
  databaseGroupList: ComposedDatabaseGroup[];
  selectedDatabaseGroupName?: string;
  selectedSchemaGroupNameList?: string[];
}>();

const emit = defineEmits<{
  (event: "update", dbGroupName: string, schemaGroupNameList: string[]): void;
}>();

const { t } = useI18n();
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  selectedDatabaseGroupName: props.selectedDatabaseGroupName,
  selectedSchemaGroupNameList: props.selectedSchemaGroupNameList || [],
});
const schemaGroupListMap = ref<Map<string, SchemaGroup[]>>(new Map());

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: "",
      width: "2rem",
    },
    { title: t("common.name"), width: "1fr" },
    {
      title: t("common.project"),
      width: "1fr",
    },
    {
      title: t("common.environment"),
      width: "1fr",
    },
  ];

  return columns;
});

const getSchemaGroupListByDBGroupName = (name: string) => {
  return schemaGroupListMap.value.get(name) || [];
};

const getSchemaGroupTitle = (name: string) => {
  const [, , title] =
    getProjectNameAndDatabaseGroupNameAndSchemaGroupName(name);
  return title || "";
};

const handleDatabaseGroupSelect = (dbGroup: ComposedDatabaseGroup) => {
  if (state.selectedDatabaseGroupName === dbGroup.name) {
    return;
  }

  state.selectedDatabaseGroupName = dbGroup.name;
  state.selectedSchemaGroupNameList = [];
  emit(
    "update",
    state.selectedDatabaseGroupName || "",
    state.selectedSchemaGroupNameList
  );
};

const handleSchemaGroupCheck = (schemaGroup: SchemaGroup) => {
  if (state.selectedSchemaGroupNameList.includes(schemaGroup.name)) {
    state.selectedSchemaGroupNameList = [];
  } else {
    // Note: now we only support one schema group.
    state.selectedSchemaGroupNameList = [schemaGroup.name];
  }
  emit(
    "update",
    state.selectedDatabaseGroupName || "",
    state.selectedSchemaGroupNameList
  );
};

watch(
  () => props,
  async () => {
    for (const dbGroup of props.databaseGroupList) {
      const schemaGroupList =
        await dbGroupStore.getOrFetchSchemaGroupListByDBGroupName(dbGroup.name);
      schemaGroupListMap.value.set(dbGroup.name, schemaGroupList);
    }
  },
  {
    immediate: true,
    deep: true,
  }
);
</script>
