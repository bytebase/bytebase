<template>
  <DrawerContent :title="$t('quick-action.transfer-in-db-title')">
    <div
      class="px-4 w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)]"
    >
      <div
        v-if="state.loading"
        class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
      >
        <BBSpin />
      </div>
      <div v-else class="space-y-4">
        <TransferSourceSelector
          :project="project"
          :raw-database-list="rawDatabaseList"
          :transfer-source="state.transferSource"
          :instance-filter="state.instanceFilter"
          :project-filter="state.projectFilter"
          :search-text="state.searchText"
          :has-permission-for-default-project="hasPermissionForDefaultProject"
          @change="state.transferSource = $event"
          @select-instance="state.instanceFilter = $event"
          @select-project="state.projectFilter = $event"
          @search-text-change="state.searchText = $event"
        />
        <MultipleDatabaseSelector
          v-if="filteredDatabaseList.length > 0"
          v-model:selected-database-name-list="state.selectedDatabaseNameList"
          :transfer-source="state.transferSource"
          :database-list="filteredDatabaseList"
        />
        <NoDataPlaceholder v-else />
      </div>
    </div>

    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <NTooltip :disabled="state.selectedDatabaseNameList.length === 0">
          <template #trigger>
            <div class="textinfolabel">
              {{
                $t("database.selected-n-databases", {
                  n: state.selectedDatabaseNameList.length,
                })
              }}
            </div>
          </template>
          <div class="mx-2">
            <ul class="list-disc">
              <li v-for="db in selectedDatabaseList" :key="db.name">
                {{ db.databaseName }}
              </li>
            </ul>
          </div>
        </NTooltip>
        <div class="flex items-center gap-x-3">
          <NButton @click.prevent="$emit('dismiss')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!allowTransfer"
            @click.prevent="transferDatabase"
          >
            {{ $t("common.transfer") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { toRef } from "vue";
import { BBSpin } from "@/bbkit";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectByName,
  useProjectV1List,
} from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { ComposedDatabase, ComposedProject } from "@/types";
import {
  DEFAULT_PROJECT_NAME,
  defaultProject,
  isValidInstanceName,
} from "@/types";
import type { UpdateDatabaseRequest } from "@/types/proto/v1/database_service";
import type { InstanceResource } from "@/types/proto/v1/instance_service";
import type { Project } from "@/types/proto/v1/project_service";
import {
  filterDatabaseV1ByKeyword,
  hasProjectPermissionV2,
  sortDatabaseV1List,
  wrapRefAsPromise,
} from "@/utils";
import NoDataPlaceholder from "../misc/NoDataPlaceholder.vue";
import { DrawerContent } from "../v2";
import MultipleDatabaseSelector from "./MultipleDatabaseSelector.vue";
import TransferSourceSelector from "./TransferSourceSelector.vue";
import type { TransferSource } from "./utils";

interface LocalState {
  transferSource: TransferSource;
  instanceFilter: InstanceResource | undefined;
  projectFilter: Project | undefined;
  searchText: string;
  loading: boolean;
  selectedDatabaseNameList: string[];
}

const props = defineProps<{
  projectName: string;
  onSuccess?: (databases: ComposedDatabase[]) => void;
}>();

const emit = defineEmits<{
  (e: "dismiss"): void;
}>();

const databaseStore = useDatabaseV1Store();
const { projectList, ready: projectListReady } = useProjectV1List();

const hasTransferDatabasePermission = (project: ComposedProject): boolean => {
  return (
    hasProjectPermissionV2(project, "bb.databases.list") &&
    hasProjectPermissionV2(project, "bb.projects.update")
  );
};

const hasPermissionForDefaultProject = computed(() => {
  return hasTransferDatabasePermission(defaultProject());
});

const state = reactive<LocalState>({
  transferSource:
    props.projectName === DEFAULT_PROJECT_NAME ||
    !hasPermissionForDefaultProject.value
      ? "OTHER"
      : "DEFAULT",
  instanceFilter: undefined,
  projectFilter: undefined,
  searchText: "",
  loading: true,
  selectedDatabaseNameList: [],
});
const { project } = useProjectByName(toRef(props, "projectName"));

watchEffect(async () => {
  if (!projectListReady) {
    return;
  }
  await Promise.all(
    projectList.value.map(async (proj) => {
      if (proj.name === props.projectName) {
        return Promise.resolve();
      }
      return await wrapRefAsPromise(useDatabaseV1List(proj.name).ready, true);
    })
  );
  state.loading = false;
});

const rawDatabaseList = computed(() => {
  if (state.transferSource === "DEFAULT") {
    return databaseStore.databaseListByProject(DEFAULT_PROJECT_NAME);
  } else {
    return databaseStore.databaseList.filter((db) => {
      return (
        db.project !== props.projectName &&
        db.project !== DEFAULT_PROJECT_NAME &&
        hasTransferDatabasePermission(db.projectEntity)
      );
    });
  }
});

const filteredDatabaseList = computed(() => {
  let list = [...rawDatabaseList.value];
  const keyword = state.searchText.trim();
  list = list.filter((db) =>
    filterDatabaseV1ByKeyword(db, keyword, [
      "name",
      "project",
      "instance",
      "environment",
    ])
  );

  list = list.filter((db) => {
    // Default uses instance filter
    if (state.transferSource === "DEFAULT") {
      const instance = state.instanceFilter;
      if (
        instance &&
        isValidInstanceName(instance.name) &&
        db.instance !== instance.name
      ) {
        return false;
      }
    }

    // Other uses project filter
    if (state.transferSource === "OTHER") {
      const project = state.projectFilter;
      if (project && db.project != project.name) {
        return false;
      }
    }

    return true;
  });

  return sortDatabaseV1List(list);
});

const allowTransfer = computed(() => state.selectedDatabaseNameList.length > 0);

const selectedDatabaseList = computed(() =>
  state.selectedDatabaseNameList.map((name) =>
    databaseStore.getDatabaseByName(name)
  )
);

const transferDatabase = async () => {
  try {
    state.loading = true;
    const updates = selectedDatabaseList.value.map((db) => {
      const databasePatch = cloneDeep(db);
      databasePatch.project = props.projectName;
      const updateMask = ["project"];
      return {
        database: databasePatch,
        updateMask,
      } as UpdateDatabaseRequest;
    });
    const updated = await databaseStore.batchUpdateDatabases({
      parent: "-",
      requests: updates,
    });
    const displayDatabaseName =
      selectedDatabaseList.value.length > 1
        ? `${selectedDatabaseList.value.length} databases`
        : `'${selectedDatabaseList.value[0].databaseName}'`;

    if (props.onSuccess) {
      props.onSuccess(updated);
    } else {
      emit("dismiss");
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Successfully transferred ${displayDatabaseName} to project '${project.value.title}'.`,
    });
  } finally {
    state.loading = false;
  }
};
</script>
