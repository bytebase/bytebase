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
    <div class="flex items-center justify-end gap-x-2">
      <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
      <NButton type="primary" :disabled="!allowConfirm" @click="doConfirm">
        {{ $t("common.confirm") }}
      </NButton>
    </div>
    <div
      v-if="loading"
      class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
    >
      <BBSpin />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from "vue";
import { uniq } from "lodash-es";
import {
  NTransfer,
  TransferRenderSourceList,
  NButton,
  NTree,
  TreeOption,
} from "naive-ui";

import {
  useDatabaseV1Store,
  useDBSchemaStore,
  useProjectV1Store,
} from "@/store";
import {
  flattenTreeOptions,
  mapTreeOptions,
  DatabaseTreeOption,
  DatabaseResource,
} from "./common";
import Label from "./Label.vue";

const props = defineProps<{
  projectId: string;
  databaseId?: string;
  selectedDatabaseResourceList: DatabaseResource[];
}>();

const emits = defineEmits<{
  (e: "close"): void;
  (e: "update", databaseResourceList: DatabaseResource[]): void;
}>();

const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaStore();
const selectedValueList = ref<string[]>([]);
const databaseResourceMap = ref<Map<string, DatabaseResource>>(new Map());
const loading = ref(false);

onMounted(async () => {
  const project = await useProjectV1Store().getOrFetchProjectByUID(
    props.projectId
  );
  const databaseList = await databaseStore.fetchDatabaseList({
    parent: "instances/-",
    filter: `project == "${project.name}"`,
  });

  for (const database of databaseList) {
    const databaseMetadata = await dbSchemaStore.getOrFetchDatabaseMetadataById(
      Number(database.uid)
    );
    databaseResourceMap.value.set(`d-${database.uid}`, {
      databaseId: database.uid,
    });
    for (const schema of databaseMetadata.schemas) {
      databaseResourceMap.value.set(`s-${database.uid}-${schema.name}`, {
        databaseId: database.uid,
        schema: schema.name,
      });
      for (const table of schema.tables) {
        databaseResourceMap.value.set(
          `t-${database.uid}-${schema.name}-${table.name}`,
          {
            databaseId: database.uid,
            schema: schema.name,
            table: table.name,
          }
        );
      }
    }
  }
  loading.value = false;
});

const databaseList = computed(() => {
  const project = useProjectV1Store().getProjectByUID(props.projectId);
  const list = databaseStore.databaseListByProject(project.name);
  return props.databaseId
    ? list.filter((item) => item.uid === props.databaseId)
    : list;
});

const allowConfirm = computed(() => {
  return selectedValueList.value.length > 0;
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

const doConfirm = () => {
  const list = selectedValueList.value
    .map((key) => {
      return databaseResourceMap.value.get(key);
    })
    .filter((item) => item !== undefined);
  emits("update", list as DatabaseResource[]);
};

watch(
  () => props.selectedDatabaseResourceList,
  () => {
    const selectedKeyList = [];
    for (const databaseResource of props.selectedDatabaseResourceList) {
      let key = "";
      if (databaseResource.table !== undefined) {
        key = `t-${databaseResource.databaseId}-${databaseResource.schema}-${databaseResource.table}`;
      } else if (databaseResource.schema !== undefined) {
        key = `s-${databaseResource.databaseId}-${databaseResource.schema}`;
      } else {
        key = `d-${databaseResource.databaseId}`;
      }
      selectedKeyList.push(key);
    }
    selectedValueList.value = uniq(selectedKeyList);
  },
  {
    immediate: true,
  }
);
</script>
