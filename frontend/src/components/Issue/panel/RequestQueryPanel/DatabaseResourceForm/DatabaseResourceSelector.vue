<template>
  <div class="w-[60rem] space-y-2">
    <NTransfer
      v-if="!loading"
      ref="transfer"
      v-model:value="selectedValueList"
      style="height: calc(100vh - 380px)"
      :options="sourceTransferOptions"
      :render-source-list="renderSourceList"
      :render-target-list="renderTargetList"
      :source-filterable="true"
      :source-filter-placeholder="$t('database.search-database-name')"
    />
  </div>
</template>

<script setup lang="ts">
import {
  NTransfer,
  TransferRenderSourceList,
  NTree,
  TreeOption,
} from "naive-ui";
import { computed, h, ref, watch } from "vue";

import {
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useProjectV1Store,
} from "@/store";
import { DatabaseResource } from "@/types";
import {
  flattenTreeOptions,
  mapTreeOptions,
  DatabaseTreeOption,
} from "./common";
import Label from "./Label.vue";

const props = defineProps<{
  projectId: string;
  databaseId?: string;
  selectedDatabaseResourceList: DatabaseResource[];
}>();

const emit = defineEmits<{
  (e: "update", databaseResourceList: DatabaseResource[]): void;
}>();

const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();

const selectedValueList = ref<string[]>(
  props.selectedDatabaseResourceList.map((databaseResource) => {
    const database = databaseStore.getDatabaseByName(
      databaseResource.databaseName
    );
    if (databaseResource.table !== undefined) {
      return `t-${database.uid}-${databaseResource.schema}-${databaseResource.table}`;
    } else if (databaseResource.schema !== undefined) {
      return `s-${database.uid}-${databaseResource.schema}`;
    } else {
      return `d-${database.uid}`;
    }
  })
);
const databaseResourceMap = ref<Map<string, DatabaseResource>>(new Map());
const loading = ref(false);

// Fetch database list when projectId changed.
watch(
  () => props.projectId,
  async () => {
    loading.value = true;
    const project = await useProjectV1Store().getOrFetchProjectByUID(
      props.projectId
    );
    const databaseList = await databaseStore.fetchDatabaseList({
      parent: "instances/-",
      filter: `project == "${project.name}"`,
    });

    for (const database of databaseList) {
      const databaseMetadata = await dbSchemaStore.getOrFetchDatabaseMetadata(
        database.name
      );
      databaseResourceMap.value.set(`d-${database.uid}`, {
        databaseName: database.name,
      });
      for (const schema of databaseMetadata.schemas) {
        databaseResourceMap.value.set(`s-${database.uid}-${schema.name}`, {
          databaseName: database.name,
          schema: schema.name,
        });
        for (const table of schema.tables) {
          databaseResourceMap.value.set(
            `t-${database.uid}-${schema.name}-${table.name}`,
            {
              databaseName: database.name,
              schema: schema.name,
              table: table.name,
            }
          );
        }
      }
    }
    loading.value = false;
  },
  {
    immediate: true,
  }
);

// Clear selectedValueList when projectId changed.
watch(
  () => props.projectId,
  () => {
    selectedValueList.value = [];
  }
);

const databaseList = computed(() => {
  const project = useProjectV1Store().getProjectByUID(props.projectId);
  const list = databaseStore.databaseListByProject(project.name);
  return props.databaseId
    ? list.filter((item) => item.uid === props.databaseId)
    : list;
});

const sourceTreeOptions = computed(() => {
  return mapTreeOptions(databaseList.value);
});

const sourceTransferOptions = computed(() => {
  const options = flattenTreeOptions(sourceTreeOptions.value);
  return options;
});

const renderSourceList: TransferRenderSourceList = ({ onCheck, pattern }) => {
  return h(NTree, {
    keyField: "value",
    checkable: true,
    selectable: false,
    checkOnClick: true,
    virtualScroll: true,
    data: sourceTreeOptions.value,
    renderLabel: ({ option }: { option: TreeOption }) => {
      return h(Label, {
        option: option as DatabaseTreeOption,
        keyword: pattern,
      });
    },
    pattern,
    checkedKeys: selectedValueList.value,
    showIrrelevantNodes: false,
    onUpdateCheckedKeys: (checkedKeys: string[]) => {
      onCheck(checkedKeys);
    },
  });
};

const targetTreeOptions = computed(() => {
  return mapTreeOptions(databaseList.value, selectedValueList.value);
});

const renderTargetList: TransferRenderSourceList = ({ onCheck }) => {
  return h(NTree, {
    keyField: "value",
    checkable: true,
    selectable: false,
    checkOnClick: true,
    virtualScroll: true,
    defaultExpandAll: true,
    data: targetTreeOptions.value,
    renderLabel: ({ option }: { option: TreeOption }) => {
      return h(Label, {
        option: option as DatabaseTreeOption,
      });
    },
    checkedKeys: selectedValueList.value,
    showIrrelevantNodes: false,
    onUpdateCheckedKeys: (checkedKeys: string[]) => {
      onCheck(checkedKeys);
    },
  });
};

watch(selectedValueList, () => {
  emit(
    "update",
    selectedValueList.value.map((key) => databaseResourceMap.value.get(key)!)
  );
});
</script>
