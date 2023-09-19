<template>
  <DrawerContent :title="$t('database.transfer-database-from-to')">
    <div
      class="w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] h-full flex flex-col gap-y-2"
    >
      <div class="flex items-center justify-between gap-x-8">
        <div class="flex-1 flex items-center gap-x-2">
          <span class="textlabel">
            {{ $t("database.transfer.source-project") }}
          </span>
          <ProjectV1Name :project="sourceProject" :link="false" />
        </div>
        <div class="flex-1 flex items-center gap-x-2">
          <span class="textlabel">
            {{ $t("database.transfer.select-target-project") }}
            <span class="text-red-500">*</span>
          </span>
          <ProjectSelect
            v-model:project="targetProjectId"
            :allowed-project-role-list="
              hasWorkspaceManageProjectPermission ? [] : [PresetRoleType.OWNER]
            "
            :include-default-project="true"
            :filter="filterTargetProject"
          />
        </div>
      </div>
      <NTransfer
        ref="transfer"
        v-model:value="selectedValueList"
        :options="sourceTransferOptions"
        :render-source-list="renderSourceList"
        :render-target-list="renderTargetList"
        :source-filterable="true"
        :source-filter-placeholder="$t('database.search-database-name')"
        class="bb-transfer-out-database-transfer"
        style="height: 100%"
      />
      <div
        class="absolute top-1/2 left-1/2 -translate-x-1/2 z-10 w-8 h-8 text-control flex items-center justify-center"
      >
        <heroicons:chevron-right class="w-8 h-8" />
      </div>
      <div
        v-if="loading"
        class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-end gap-x-3">
        <NButton @click="$emit('dismiss')">{{ $t("common.cancel") }}</NButton>
        <NTooltip :disabled="allowTransfer">
          <template #trigger>
            <NButton
              type="primary"
              :disabled="!allowTransfer"
              tag="div"
              @click="doTransfer"
            >
              {{ $t("common.transfer") }}
            </NButton>
          </template>
          <ul>
            <li v-for="(error, i) in validationErrors" :key="i">
              {{ error }}
            </li>
          </ul>
        </NTooltip>
      </div>
    </template>
  </DrawerContent>
</template>

<script setup lang="ts">
import { cloneDeep } from "lodash-es";
import {
  NTransfer,
  NTree,
  TreeOption,
  TransferRenderSourceList,
  NButton,
  NTooltip,
} from "naive-ui";
import { computed, h, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import { ProjectV1Name, ProjectSelect, DrawerContent } from "@/components/v2";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useGracefulRequest,
  useProjectV1ByUID,
  useProjectV1Store,
} from "@/store";
import { ComposedDatabase, PresetRoleType, UNKNOWN_ID } from "@/types";
import { Project } from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import Label from "./Label.vue";
import {
  DatabaseTreeOption,
  flattenTreeOptions,
  mapTreeOptions,
} from "./common";

const props = defineProps<{
  projectId: string;
}>();

const emit = defineEmits<{
  (e: "dismiss"): void;
}>();

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const loading = ref(false);
const transfer = ref<InstanceType<typeof NTransfer>>();

const hasWorkspaceManageProjectPermission = computed(() =>
  hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-project",
    currentUser.value.userRole
  )
);

const databaseList = computed(() => {
  const project = projectStore.getProjectByUID(props.projectId);
  return databaseStore.databaseListByProject(project.name);
});

const selectedValueList = ref<string[]>([]);
const selectedDatabaseList = computed(() => {
  return selectedValueList.value
    .filter((value) => value.startsWith("database-"))
    .map((value) => {
      const uid = value.split("-").pop()!;
      return databaseStore.getDatabaseByUID(uid);
    });
});
const targetProjectId = ref<string>();
const targetProject = computed(() => {
  const id = targetProjectId.value;
  if (!id || id === String(UNKNOWN_ID)) return undefined;
  return projectStore.getProjectByUID(id);
});

const { project: sourceProject } = useProjectV1ByUID(toRef(props, "projectId"));

const validationErrors = computed(() => {
  const errors: string[] = [];
  if (!targetProject.value) {
    errors.push(t("database.transfer.errors.select-target-project"));
  }
  if (selectedDatabaseList.value.length === 0) {
    errors.push(t("database.transfer.errors.select-at-least-one-database"));
  }
  return errors;
});

const allowTransfer = computed(() => {
  return validationErrors.value.length === 0;
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
  return project.uid !== props.projectId;
};

const doTransfer = async () => {
  const target = targetProject.value!;
  if (!target) return;

  const transferOneDatabase = async (database: ComposedDatabase) => {
    const databasePatch = cloneDeep(database);
    databasePatch.project = target.name;
    const updateMask = ["project"];
    const updated = await useDatabaseV1Store().updateDatabase({
      database: databasePatch,
      updateMask,
    });
    return updated;
  };

  const databaseList = selectedDatabaseList.value;

  try {
    loading.value = true;
    await useGracefulRequest(async () => {
      const requests = databaseList.map((db) => {
        transferOneDatabase(db);
      });
      await Promise.all(requests);

      const displayDatabaseName =
        databaseList.length > 1
          ? `${databaseList.length} databases`
          : `'${databaseList[0].databaseName}'`;

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: `Successfully transferred ${displayDatabaseName} to project '${target.title}'.`,
      });
      emit("dismiss");
    });
  } finally {
    loading.value = false;
  }
};
</script>

<style>
.bb-transfer-out-database-transfer {
  @apply gap-x-8;
}

.bb-transfer-out-database-transfer .n-transfer-list--target {
  border-left: 1px solid var(--n-border-color) !important;
}
</style>
