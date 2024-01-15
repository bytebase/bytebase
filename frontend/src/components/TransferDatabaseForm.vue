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
          @change="state.transferSource = $event"
          @select-instance="state.instanceFilter = $event"
          @select-project="state.projectFilter = $event"
          @search-text-change="state.searchText = $event"
        />
        <MultipleDatabaseSelector
          v-if="filteredDatabaseList.length > 0"
          v-model:selected-uid-list="state.selectedDatabaseUidList"
          :transfer-source="state.transferSource"
          :database-list="filteredDatabaseList"
        />
        <NoDataPlaceholder v-else />
      </div>
    </div>

    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <div>
          <div
            v-if="state.selectedDatabaseUidList.length > 0"
            class="textinfolabel"
          >
            {{
              $t("database.selected-n-databases", {
                n: state.selectedDatabaseUidList.length,
              })
            }}
          </div>
        </div>
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
import { computed, onBeforeMount, reactive } from "vue";
import { toRef } from "vue";
import {
  MultipleDatabaseSelector,
  TransferSource,
  TransferSourceSelector,
} from "@/components/TransferDatabaseForm";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1ByUID,
  useProjectV1Store,
} from "@/store";
import { Project } from "@/types/proto/v1/project_service";
import {
  DEFAULT_PROJECT_ID,
  ComposedInstance,
  DEFAULT_PROJECT_V1_NAME,
  UNKNOWN_INSTANCE_NAME,
  ComposedDatabase,
} from "../types";
import {
  filterDatabaseV1ByKeyword,
  hasProjectPermissionV2,
  sortDatabaseV1List,
} from "../utils";

interface LocalState {
  transferSource: TransferSource;
  instanceFilter: ComposedInstance | undefined;
  projectFilter: Project | undefined;
  searchText: string;
  loading: boolean;
  selectedDatabaseUidList: string[];
}

const props = defineProps({
  projectId: {
    required: true,
    type: String,
  },
});

const emit = defineEmits<{
  (e: "dismiss"): void;
}>();

const currentUserV1 = useCurrentUserV1();
const databaseStore = useDatabaseV1Store();

const state = reactive<LocalState>({
  transferSource:
    props.projectId === String(DEFAULT_PROJECT_ID) ? "OTHER" : "DEFAULT",
  instanceFilter: undefined,
  projectFilter: undefined,
  searchText: "",
  loading: false,
  selectedDatabaseUidList: [],
});
const hasWorkspaceManageDatabasePermission = computed(() => {
  return hasProjectPermissionV2(
    project.value,
    currentUserV1.value,
    "bb.projects.update"
  );
});
const { project } = useProjectV1ByUID(toRef(props, "projectId"));

const prepare = async () => {
  await databaseStore.fetchDatabaseList({
    parent: "instances/-",
  });
};

onBeforeMount(prepare);

const rawDatabaseList = computed(() => {
  if (state.transferSource === "DEFAULT") {
    return databaseStore.databaseListByProject(DEFAULT_PROJECT_V1_NAME);
  } else {
    const rawList = hasWorkspaceManageDatabasePermission.value
      ? databaseStore.databaseList
      : databaseStore.databaseListByUser(currentUserV1.value);

    return [...rawList].filter(
      (item) =>
        item.projectEntity.uid !== props.projectId &&
        item.project !== DEFAULT_PROJECT_V1_NAME
    );
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
        instance.name !== UNKNOWN_INSTANCE_NAME &&
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

const allowTransfer = computed(() => state.selectedDatabaseUidList.length > 0);

const transferDatabase = async () => {
  const databaseList = state.selectedDatabaseUidList.map((uid) =>
    databaseStore.getDatabaseByUID(uid)
  );
  const transferOneDatabase = async (database: ComposedDatabase) => {
    const targetProject = useProjectV1Store().getProjectByUID(props.projectId);
    const databasePatch = cloneDeep(database);
    databasePatch.project = targetProject.name;
    const updateMask = ["project"];
    const updated = await useDatabaseV1Store().updateDatabase({
      database: databasePatch,
      updateMask,
    });
    return updated;
  };

  try {
    state.loading = true;
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
      title: `Successfully transferred ${displayDatabaseName} to project '${project.value.title}'.`,
    });
    emit("dismiss");
  } finally {
    state.loading = false;
  }
};
</script>
