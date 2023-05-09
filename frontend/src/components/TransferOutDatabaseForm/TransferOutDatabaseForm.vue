<template>
  <div class="w-[60rem] space-y-2">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-x-2">
        <label class="textlabel">
          {{ $t("database.transfer.source-project") }}
        </label>
        <ProjectName :project="sourceProject" :link="false" />
      </div>
      <div class="flex items-center gap-x-2">
        <label class="textlabel">
          {{ $t("database.transfer.target-project") }}
        </label>
        <ProjectSelect
          v-model:project="targetProjectId"
          :allowed-project-role-list="['OWNER']"
          :include-default-project="true"
          :filter="filterTargetProject"
        />
      </div>
    </div>
    <NTransfer
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
      <NButton @click="$emit('dismiss')">{{ $t("common.cancel") }}</NButton>
      <NButton type="primary" :disabled="!allowTransfer" @click="doTransfer">
        {{ $t("common.transfer") }}
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
import { computed, h, ref } from "vue";
import {
  NTransfer,
  NTree,
  TreeOption,
  TransferRenderSourceList,
  NButton,
} from "naive-ui";

import { Database, IdType, Project, ProjectId, UNKNOWN_ID } from "@/types";
import { pushNotification, useDatabaseStore, useProjectStore } from "@/store";
import { ProjectName, ProjectSelect } from "../v2";
import Label from "./Label.vue";
import {
  DatabaseTreeOption,
  flattenTreeOptions,
  mapTreeOptions,
} from "./common";

const props = defineProps<{
  projectId: ProjectId;
}>();

const emit = defineEmits<{
  (e: "dismiss"): void;
}>();

const projectStore = useProjectStore();
const databaseStore = useDatabaseStore();
const loading = ref(false);
const transfer = ref<InstanceType<typeof NTransfer>>();

const databaseList = computed(() => {
  return databaseStore.getDatabaseListByProjectId(props.projectId);
});

const selectedValueList = ref<string[]>([]);
const selectedDatabaseList = computed(() => {
  return selectedValueList.value
    .filter((value) => value.startsWith("database-"))
    .map((value) => {
      const id = parseInt(value.split("-").pop()!, 10);
      return databaseStore.getDatabaseById(id);
    });
});
const targetProjectId = ref<IdType>();
const targetProject = computed(() => {
  const id = targetProjectId.value;
  if (!id || id === UNKNOWN_ID) return undefined;
  return projectStore.getProjectById(id);
});

const sourceProject = computed(() => {
  return useProjectStore().getProjectById(props.projectId);
});

const allowTransfer = computed(() => {
  if (!targetProject.value) return false;
  return selectedDatabaseList.value.length > 0;
});

const sourceTreeOptions = computed(() => {
  return mapTreeOptions(databaseList.value);
});
const sourceTransferOptions = computed(() => {
  return flattenTreeOptions(sourceTreeOptions.value);
});

const renderSourceList: TransferRenderSourceList = ({ onCheck, pattern }) => {
  return h(NTree, {
    style: "margin: 0 4px;",
    keyField: "value",
    defaultExpandAll: true,
    checkable: true,
    selectable: false,
    blockLine: true,
    checkOnClick: true,
    cascade: true,
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
      onCheck(checkedKeys.filter((value) => value.startsWith("database-")));
    },
  });
};

const targetTreeOptions = computed(() => {
  return mapTreeOptions(selectedDatabaseList.value);
});
const targetCheckedKeys = computed(() => {
  return flattenTreeOptions(targetTreeOptions.value).map(
    (option) => option.value
  );
});

const renderTargetList: TransferRenderSourceList = ({ onCheck }) => {
  return h(NTree, {
    key: targetCheckedKeys.value.join(","),
    style: "margin: 0 4px;",
    keyField: "value",
    defaultExpandAll: true,
    selectable: false,
    blockLine: true,
    checkable: true,
    checkOnClick: true,
    cascade: true,
    virtualScroll: true,
    data: targetTreeOptions.value,
    renderLabel: ({ option }: { option: TreeOption }) => {
      return h(Label, {
        option: option as DatabaseTreeOption,
      });
    },
    checkedKeys: targetCheckedKeys.value,
    onUpdateCheckedKeys: (checkedKeys: string[]) => {
      onCheck(checkedKeys.filter((value) => value.startsWith("database-")));
    },
  });
};

const filterTargetProject = (project: Project) => {
  return project.id !== props.projectId;
};

const doTransfer = async () => {
  const target = targetProject.value!;
  if (!target) if (!targetProject.value) return;

  const transferOneDatabase = (database: Database) => {
    return databaseStore.transferProject({
      databaseId: database.id,
      projectId: target.id,
    });
  };

  const databaseList = selectedDatabaseList.value;

  try {
    loading.value = true;
    const requests = databaseList.map((db) => {
      transferOneDatabase(db);
    });
    await new Promise((resolve) => setTimeout(resolve, 2000));
    await Promise.all(requests);
    const displayDatabaseName =
      databaseList.length > 1
        ? `${databaseList.length} databases`
        : `'${databaseList[0].name}'`;

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Successfully transferred ${displayDatabaseName} to project '${target.name}'.`,
    });
    emit("dismiss");
  } finally {
    loading.value = false;
  }
};
</script>
