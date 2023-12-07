<template>
  <div class="flex flex-col relative space-y-4">
    <AdvancedSearchBox
      v-model:params="state.params"
      class="px-4"
      :autofocus="false"
      :placeholder="$t('database.filter-database')"
      :support-option-id-list="supportOptionIdList"
    />

    <DatabaseOperations
      v-if="selectedDatabases.length > 0 || isStandaloneMode"
      class="mb-3"
      :databases="selectedDatabases"
      @dismiss="state.selectedDatabaseIds.clear()"
    />

    <DatabaseV1Table
      pagination-class="mb-4"
      table-class="border-y"
      :database-list="filteredDatabaseList"
      :database-group-list="filteredDatabaseGroupList"
      :show-placeholder="true"
      :show-selection-column="true"
      :custom-click="isStandaloneMode"
      @select-database="(db: ComposedDatabase) =>
                  toggleDatabasesSelection([db as ComposedDatabase], !isDatabaseSelected(db))"
    >
      <template #selection-all="{ databaseList }">
        <input
          v-if="databaseList.length > 0"
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          v-bind="getAllSelectionState(databaseList)"
          @input="
            toggleDatabasesSelection(
              databaseList,
              ($event.target as HTMLInputElement).checked
            )
          "
        />
      </template>
      <template #selection="{ database }">
        <input
          v-if="isDatabase(database)"
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          :checked="isDatabaseSelected(database as ComposedDatabase)"
          @click.stop="
            toggleDatabasesSelection(
              [database],
              ($event.target as HTMLInputElement).checked
            )
          "
        />
        <div v-else class="text-control-light cursor-not-allowed ml-auto">
          -
        </div>
      </template>
    </DatabaseV1Table>

    <div
      v-if="state.loading"
      class="absolute inset-0 bg-white/50 flex justify-center items-center"
    >
      <BBSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect, onMounted, reactive, ref } from "vue";
import { DatabaseV1Table } from "@/components/v2";
import { isDatabase } from "@/components/v2/Model/DatabaseV1Table/utils";
import {
  useCurrentUserV1,
  useDBGroupStore,
  useDatabaseV1Store,
  usePageMode,
  usePolicyV1Store,
  useProjectV1ListByCurrentUser,
  useUIStateStore,
} from "@/store";
import {
  UNKNOWN_ID,
  UNKNOWN_USER_NAME,
  ComposedDatabase,
  ComposedDatabaseGroup,
  DEFAULT_PROJECT_V1_NAME,
} from "@/types";
import {
  Policy,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import {
  SearchParams,
  filterDatabaseV1ByKeyword,
  sortDatabaseV1List,
  isDatabaseV1Accessible,
  CommonFilterScopeIdList,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
} from "@/utils";

interface LocalState {
  databaseGroupList: ComposedDatabaseGroup[];
  loading: boolean;
  selectedDatabaseIds: Set<string>;
  params: SearchParams;
}

const uiStateStore = useUIStateStore();
const { projectList } = useProjectV1ListByCurrentUser();
const pageMode = usePageMode();

const state = reactive<LocalState>({
  databaseGroupList: [],
  loading: false,
  selectedDatabaseIds: new Set(),
  params: {
    query: "",
    scopes: [],
  },
});

const currentUserV1 = useCurrentUserV1();
const databaseV1Store = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const policyList = ref<Policy[]>([]);

const preparePolicyList = () => {
  usePolicyV1Store()
    .fetchPolicies({
      policyType: PolicyType.WORKSPACE_IAM,
      resourceType: PolicyResourceType.WORKSPACE,
    })
    .then((list) => (policyList.value = list));
};

watchEffect(preparePolicyList);

const isStandaloneMode = computed(() => pageMode.value === "STANDALONE");

const selectedInstance = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "instance")?.value ??
    `${UNKNOWN_ID}`
  );
});

const selectedEnvironment = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "environment")?.value ??
    `${UNKNOWN_ID}`
  );
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("database.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "database.visit",
      newState: true,
    });
  }
});

const prepareDatabaseList = async () => {
  // It will also be called when user logout
  if (currentUserV1.value.name !== UNKNOWN_USER_NAME) {
    state.loading = true;
    await databaseV1Store.fetchDatabaseList({
      parent: "instances/-",
    });
    state.loading = false;
  }
};

const databaseV1List = computed(() => {
  return sortDatabaseV1List(databaseV1Store.databaseList).filter((db) =>
    projectList.value.map((project) => project.name).includes(db.project)
  );
});

const prepareDatabaseGroupList = async () => {
  if (currentUserV1.value.name !== UNKNOWN_USER_NAME) {
    state.databaseGroupList = (
      await dbGroupStore.fetchAllDatabaseGroupList()
    ).filter((dbGroup) =>
      projectList.value
        .map((project) => project.name)
        .includes(dbGroup.project.name)
    );
  }
};

watchEffect(async () => {
  state.loading = true;
  await prepareDatabaseList();
  await prepareDatabaseGroupList();
  state.loading = false;
});

const filteredDatabaseList = computed(() => {
  let list = databaseV1List.value.filter((database) =>
    isDatabaseV1Accessible(database, currentUserV1.value)
  );
  if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) =>
        extractEnvironmentResourceName(db.effectiveEnvironment) ===
        selectedEnvironment.value
    );
  }
  if (selectedInstance.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) =>
        extractInstanceResourceName(db.instanceEntity.name) ===
        selectedInstance.value
    );
  }
  if (isStandaloneMode.value) {
    list = list.filter(
      (db) => db.projectEntity.name !== DEFAULT_PROJECT_V1_NAME
    );
  }
  const keyword = state.params.query.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
        "project",
      ])
    );
  }
  return list;
});

const filteredDatabaseGroupList = computed(() => {
  let list = [...state.databaseGroupList];
  if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (dbGroup) =>
        extractEnvironmentResourceName(dbGroup.environmentName) ===
        selectedEnvironment.value
    );
  }
  const keyword = state.params.query.trim().toLowerCase();
  if (keyword) {
    list = list.filter(
      (dbGroup) =>
        dbGroup.name.includes(keyword) ||
        dbGroup.databasePlaceholder.includes(keyword)
    );
  }
  return list;
});

const getAllSelectionState = (
  databaseList: (ComposedDatabase | ComposedDatabaseGroup)[]
): { checked: boolean; indeterminate: boolean } => {
  const filteredDatabases = databaseList.filter((db) =>
    isDatabase(db)
  ) as ComposedDatabase[];

  const checked =
    state.selectedDatabaseIds.size > 0 &&
    filteredDatabases.every((db) => state.selectedDatabaseIds.has(db.uid));
  const indeterminate =
    !checked &&
    filteredDatabases.some((db) => state.selectedDatabaseIds.has(db.uid));

  return {
    checked,
    indeterminate,
  };
};

const toggleDatabasesSelection = (
  databaseList: (ComposedDatabase | ComposedDatabaseGroup)[],
  on: boolean
): void => {
  if (on) {
    databaseList.forEach((db) => {
      if (isDatabase(db)) {
        state.selectedDatabaseIds.add((db as ComposedDatabase).uid);
      }
    });
  } else {
    databaseList.forEach((db) => {
      if (isDatabase(db)) {
        state.selectedDatabaseIds.delete((db as ComposedDatabase).uid);
      }
    });
  }
};

const isDatabaseSelected = (database: ComposedDatabase): boolean => {
  return state.selectedDatabaseIds.has((database as ComposedDatabase).uid);
};

const selectedDatabases = computed((): ComposedDatabase[] => {
  return filteredDatabaseList.value.filter(
    (db) => isDatabase(db) && state.selectedDatabaseIds.has(db.uid)
  );
});

const supportOptionIdList = computed(() => [...CommonFilterScopeIdList]);
</script>
