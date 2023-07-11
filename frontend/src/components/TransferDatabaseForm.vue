<template>
  <TransferMultipleDatabaseForm
    :target-project="project"
    :transfer-source="state.transferSource"
    :database-list="filteredDatabaseList"
    @dismiss="$emit('dismiss')"
    @submit="(databaseList) => transferDatabase(databaseList)"
  >
    <template #transfer-source-selector>
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
    </template>
  </TransferMultipleDatabaseForm>

  <div
    v-if="state.loading"
    class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
  >
    <BBSpin />
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, reactive } from "vue";
import { cloneDeep } from "lodash-es";
import {
  TransferMultipleDatabaseForm,
  TransferSource,
  TransferSourceSelector,
} from "@/components/TransferDatabaseForm";
import {
  DEFAULT_PROJECT_ID,
  ComposedInstance,
  DEFAULT_PROJECT_V1_NAME,
  UNKNOWN_INSTANCE_NAME,
  ComposedDatabase,
} from "../types";
import {
  buildDatabaseNameRegExpByTemplate,
  filterDatabaseV1ByKeyword,
  PRESET_LABEL_KEY_PLACEHOLDERS,
  sortDatabaseV1List,
  useWorkspacePermissionV1,
} from "../utils";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1ByUID,
  useProjectV1Store,
} from "@/store";
import { toRef } from "vue";
import { Project } from "@/types/proto/v1/project_service";

interface LocalState {
  transferSource: TransferSource;
  instanceFilter: ComposedInstance | undefined;
  projectFilter: Project | undefined;
  searchText: string;
  loading: boolean;
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
});
const hasWorkspaceManageDatabasePermission = useWorkspacePermissionV1(
  "bb.permission.workspace.manage-database"
);
const { project } = useProjectV1ByUID(toRef(props, "projectId"));

const prepare = async () => {
  await databaseStore.searchDatabaseList({
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

const transferDatabase = async (databaseList: ComposedDatabase[]) => {
  const transferOneDatabase = async (
    database: ComposedDatabase,
    labels?: Record<string, string>
  ) => {
    const targetProject = useProjectV1Store().getProjectByUID(props.projectId);
    const databasePatch = cloneDeep(database);
    databasePatch.project = targetProject.name;
    const updateMask = ["project"];
    if (labels) {
      databasePatch.labels = labels;
      updateMask.push("labels");
    }
    const updated = await useDatabaseV1Store().updateDatabase({
      database: databasePatch,
      updateMask,
    });
    return updated;
  };

  try {
    state.loading = true;
    const requests = databaseList.map((db) => {
      const labels = parseLabelsIfNeeded(db);
      transferOneDatabase(db, labels);
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

const parseLabelsIfNeeded = (database: ComposedDatabase) => {
  const { dbNameTemplate } = project.value;
  if (!dbNameTemplate) return undefined;

  const regex = buildDatabaseNameRegExpByTemplate(dbNameTemplate);
  const match = database.name.match(regex);
  if (!match) return undefined;

  const labels: Record<string, string> = {
    "bb.environment": database.instanceEntity.environment,
  };

  PRESET_LABEL_KEY_PLACEHOLDERS.forEach(([placeholder, key]) => {
    const value = match.groups?.[placeholder];
    if (value) {
      labels[key] = value;
    }
  });

  return labels;
};
</script>
