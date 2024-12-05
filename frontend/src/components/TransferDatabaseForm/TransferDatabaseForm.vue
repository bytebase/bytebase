<template>
  <DrawerContent :title="$t('quick-action.transfer-in-db-title')">
    <div
      class="px-4 w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)]"
    >
      <div class="space-y-4">
        <TransferSourceSelector
          v-model:transfer-source="state.transferSource"
          :project="project"
          :raw-database-list="rawDatabaseList"
          :environment-filter="state.environmentFilter"
          :instance-filter="state.instanceFilter"
          :search-text="state.searchText"
          :has-permission-for-default-project="hasPermissionForDefaultProject"
          @select-environment="state.environmentFilter = $event"
          @select-instance="state.instanceFilter = $event"
          @search-text-change="state.searchText = $event"
        />
        <div>
          <ProjectSelect
            v-if="state.transferSource == 'OTHER'"
            class="!w-48"
            :include-all="false"
            :project-name="state.fromProjectName"
            :filter="filterSourceProject"
            @update:project-name="changeProjectFilter"
          />
        </div>
        <template
          v-if="state.transferSource === 'OTHER' && !state.fromProjectName"
        >
          <!-- Empty -->
        </template>
        <template v-else>
          <div class="w-full relative">
            <div
              v-if="state.loading"
              class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
            >
              <BBSpin />
            </div>
            <template v-else>
              <DatabaseV1Table
                mode="PROJECT"
                :database-list="filteredDatabaseList"
                :show-selection="true"
                :selected-database-names="state.selectedDatabaseNameList"
                @update:selected-databases="
                  state.selectedDatabaseNameList = Array.from($event)
                "
              />
            </template>
          </div>
        </template>
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
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { toRef } from "vue";
import { BBSpin } from "@/bbkit";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectByName,
} from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { ComposedDatabase, ComposedProject } from "@/types";
import {
  DEFAULT_PROJECT_NAME,
  defaultProject,
  isValidProjectName,
} from "@/types";
import type { Environment } from "@/types/proto/v1/environment_service";
import type { InstanceResource } from "@/types/proto/v1/instance_service";
import {
  filterDatabaseV1ByKeyword,
  hasProjectPermissionV2,
  sortDatabaseV1List,
  wrapRefAsPromise,
} from "@/utils";
import { DrawerContent, ProjectSelect } from "../v2";
import DatabaseV1Table from "../v2/Model/DatabaseV1Table/DatabaseV1Table.vue";
import TransferSourceSelector from "./TransferSourceSelector.vue";
import type { TransferSource } from "./utils";

interface LocalState {
  transferSource: TransferSource;
  instanceFilter: InstanceResource | undefined;
  environmentFilter: Environment | undefined;
  searchText: string;
  loading: boolean;
  selectedDatabaseNameList: string[];
  fromProjectName: string | undefined;
}

const props = defineProps<{
  projectName: string;
  onSuccess?: (databases: ComposedDatabase[]) => void;
}>();

const emit = defineEmits<{
  (e: "dismiss"): void;
}>();

const databaseStore = useDatabaseV1Store();

const state = reactive<LocalState>({
  transferSource: "DEFAULT",
  instanceFilter: undefined,
  environmentFilter: undefined,
  searchText: "",
  selectedDatabaseNameList: [],
  fromProjectName: undefined,
  loading: false,
});
const { project } = useProjectByName(toRef(props, "projectName"));

watchEffect(async () => {
  if (state.transferSource === "OTHER" && !state.fromProjectName) {
    return;
  }

  let fetchingProject = DEFAULT_PROJECT_NAME;
  if (state.fromProjectName) {
    fetchingProject = state.fromProjectName;
  }
  state.loading = true;
  await wrapRefAsPromise(useDatabaseV1List(fetchingProject).ready, true);
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
    const environment = state.environmentFilter;
    if (environment && db.effectiveEnvironment !== environment.name) {
      return false;
    }
    const instance = state.instanceFilter;
    if (instance && db.instance !== instance.name) {
      return false;
    }

    // Other uses project filter
    if (state.transferSource === "OTHER") {
      if (state.fromProjectName && db.project !== state.fromProjectName) {
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

const hasTransferDatabasePermission = (project: ComposedProject): boolean => {
  return (
    hasProjectPermissionV2(project, "bb.databases.list") &&
    hasProjectPermissionV2(project, "bb.projects.update")
  );
};

const hasPermissionForDefaultProject = computed(() => {
  return hasTransferDatabasePermission(defaultProject());
});

watchEffect(() => {
  if (!hasPermissionForDefaultProject.value) {
    state.transferSource = "OTHER";
  }
});

const changeProjectFilter = (name: string | undefined) => {
  if (!name || !isValidProjectName(name)) {
    state.fromProjectName = undefined;
  } else {
    state.fromProjectName = name;
  }
};

const filterSourceProject = (project: ComposedProject) => {
  return (
    hasTransferDatabasePermission(project) && project.name !== props.projectName
  );
};

const transferDatabase = async () => {
  try {
    state.loading = true;

    const updated = await useDatabaseV1Store().transferDatabases(
      selectedDatabaseList.value,
      props.projectName
    );

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
