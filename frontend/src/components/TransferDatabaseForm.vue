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
        :search-text="state.searchText"
        @change="state.transferSource = $event"
        @select-instance="state.instanceFilter = $event"
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
  Database,
  DEFAULT_PROJECT_ID,
  DatabaseLabel,
  Instance,
  UNKNOWN_ID,
} from "../types";
import {
  buildDatabaseNameRegExpByTemplate,
  filterDatabaseByKeyword,
  PRESET_LABEL_KEY_PLACEHOLDERS,
  sortDatabaseListByEnvironmentV1,
  useWorkspacePermissionV1,
} from "../utils";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseStore,
  useEnvironmentV1List,
  useProjectV1ByUID,
} from "@/store";
import { toRef } from "vue";

interface LocalState {
  transferSource: TransferSource;
  instanceFilter: Instance | undefined;
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

const databaseStore = useDatabaseStore();
const currentUserV1 = useCurrentUserV1();

const state = reactive<LocalState>({
  transferSource:
    props.projectId === String(DEFAULT_PROJECT_ID) ? "OTHER" : "DEFAULT",
  instanceFilter: undefined,
  searchText: "",
  loading: false,
});
const hasWorkspaceManageDatabasePermission = useWorkspacePermissionV1(
  "bb.permission.workspace.manage-database"
);
const { project } = useProjectV1ByUID(toRef(props, "projectId"));

const prepare = async () => {
  await databaseStore.fetchDatabaseListByProjectId(String(DEFAULT_PROJECT_ID));
};

onBeforeMount(prepare);

const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const rawDatabaseList = computed(() => {
  if (state.transferSource == "DEFAULT") {
    return cloneDeep(
      databaseStore.getDatabaseListByProjectId(String(DEFAULT_PROJECT_ID))
    );
  } else {
    const list = hasWorkspaceManageDatabasePermission.value
      ? databaseStore.getDatabaseList()
      : databaseStore.getDatabaseListByUser(currentUserV1.value);
    return cloneDeep(list).filter(
      (item: Database) =>
        String(item.project.id) !== props.projectId &&
        String(item.project.id) !== String(DEFAULT_PROJECT_ID)
    );
  }
});

const filteredDatabaseList = computed(() => {
  let list = [...rawDatabaseList.value];
  const keyword = state.searchText.trim();
  list = list.filter((db) =>
    filterDatabaseByKeyword(db, keyword, [
      "name",
      "project",
      "instance",
      "environment",
    ])
  );

  if (state.instanceFilter && state.instanceFilter.id !== UNKNOWN_ID) {
    list = list.filter((db) => db.instance.id === state.instanceFilter?.id);
  }

  return sortDatabaseListByEnvironmentV1(list, environmentList.value);
});

const transferDatabase = async (databaseList: Database[]) => {
  const transferOneDatabase = (
    database: Database,
    labels?: DatabaseLabel[]
  ) => {
    return databaseStore.transferProject({
      database,
      projectId: props.projectId,
      labels, // Will keep all labels if not specified here
    });
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
        : `'${databaseList[0].name}'`;

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

const parseLabelsIfNeeded = (
  database: Database
): DatabaseLabel[] | undefined => {
  const { dbNameTemplate } = project.value;
  if (!dbNameTemplate) return undefined;

  const regex = buildDatabaseNameRegExpByTemplate(dbNameTemplate);
  const match = database.name.match(regex);
  if (!match) return undefined;

  const environmentLabel: DatabaseLabel = {
    key: "bb.environment",
    value: database.instance.environment.name,
  };
  const parsedLabelList: DatabaseLabel[] = [];
  PRESET_LABEL_KEY_PLACEHOLDERS.forEach(([placeholder, key]) => {
    const value = match.groups?.[placeholder];
    if (value) {
      parsedLabelList.push({ key, value });
    }
  });

  return [environmentLabel, ...parsedLabelList];
};
</script>
