<template>
  <DrawerContent>
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>{{
          isEditSchema ? $t("database.edit-schema") : $t("database.change-data")
        }}</span>
        <i18n-t
          v-if="isTenantProject"
          class="text-sm textinfolabel"
          tag="span"
          keypath="deployment-config.pipeline-generated-from-deployment-config"
        >
          <template #deployment_config>
            <router-link
              :to="`/project/${projectV1Slug(state.project!)}#databases`"
              class="underline hover:bg-link-hover"
              active-class=""
              exact-active-class=""
            >
              {{ $t("common.deployment-config") }}
            </router-link>
          </template>
        </i18n-t>
      </div>
    </template>

    <div
      class="space-y-4 w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div v-if="ready">
        <template v-if="projectId">
          <!-- tenant mode project -->
          <template v-if="isTenantProject">
            <NTabs v-model:value="state.alterType">
              <NTabPane :tab="$t('alter-schema.alter-db-group')" name="TENANT">
                <div>
                  <ProjectTenantView
                    :state="state"
                    :database-list="selectableDatabaseList"
                    :environment-list="environmentList"
                    :project="state.project"
                    @dismiss="cancel"
                  />
                  <SchemalessDatabaseTable
                    v-if="isEditSchema"
                    mode="PROJECT"
                    :database-list="schemalessDatabaseList"
                  />
                </div>
              </NTabPane>
              <NTabPane
                :tab="$t('alter-schema.alter-multiple-db')"
                name="MULTI_DB"
              >
                <div class="px-1 space-x-2 mb-4">
                  <NRadio
                    :checked="state.databaseSelectedTab === 'DATABASE'"
                    value="DATABASE"
                    name="database-tab"
                    @update:checked="state.databaseSelectedTab = 'DATABASE'"
                  >
                    {{ $t("common.database") }}
                  </NRadio>
                  <NRadio
                    :checked="state.databaseSelectedTab === 'DATABASE_GROUP'"
                    value="DATABASE_GROUP"
                    name="database-tab"
                    @update:checked="handleDatabaseGroupTabSelect"
                  >
                    <div class="flex flex-row items-center">
                      <span class="mr-1">{{ $t("database-group.self") }}</span>
                      <FeatureBadge feature="bb.feature.database-grouping" />
                    </div>
                  </NRadio>
                </div>
                <div v-if="state.databaseSelectedTab === 'DATABASE'">
                  <DatabaseV1Table
                    mode="PROJECT_SHORT"
                    table-class="border"
                    :custom-click="true"
                    :database-list="selectableDatabaseList"
                    :show-selection-column="true"
                    @select-database="
                      (db: ComposedDatabase) =>
                        toggleDatabasesSelection([db as ComposedDatabase], !isDatabaseSelected(db))
                    "
                  >
                    <template
                      #selection-all="{ databaseList: renderedDatabaseList }"
                    >
                      <input
                        v-if="renderedDatabaseList.length > 0"
                        type="checkbox"
                        class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                        v-bind="getAllSelectionState(renderedDatabaseList as ComposedDatabase[])"
                        @input="
                          toggleDatabasesSelection(
                            renderedDatabaseList as ComposedDatabase[],
                            ($event.target as HTMLInputElement).checked
                          )
                        "
                      />
                    </template>
                    <template #selection="{ database }">
                      <input
                        type="checkbox"
                        class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                        :checked="isDatabaseSelected(database as ComposedDatabase)"
                        @click.stop="
                          toggleDatabasesSelection(
                            [database as ComposedDatabase],
                            ($event.target as HTMLInputElement).checked
                          )
                        "
                      />
                    </template>
                  </DatabaseV1Table>
                  <SchemalessDatabaseTable
                    v-if="isEditSchema"
                    mode="PROJECT"
                    :database-list="schemalessDatabaseList"
                  />
                </div>
                <div v-else-if="state.databaseSelectedTab === 'DATABASE_GROUP'">
                  <SelectDatabaseGroupTable
                    :show-selection="true"
                    :database-group-list="databaseGroupList"
                    :selected-database-group-name="
                      state.selectedDatabaseGroupName
                    "
                    @update="selectDatabaseGroup"
                  />
                </div>
              </NTabPane>
              <template #suffix>
                <NInputGroup class="py-0.5">
                  <template v-if="state.alterType === 'TENANT'">
                    <NInputGroupLabel
                      :bordered="false"
                      style="--n-group-label-color: transparent"
                    >
                      Group by
                    </NInputGroupLabel>
                    <YAxisRadioGroup
                      v-model:label="state.label"
                      :database-list="filteredDatabaseList"
                    />
                  </template>
                  <DatabaseLabelFilter
                    v-model:selected="state.selectedLabels"
                    :database-list="rawDatabaseList"
                  />
                  <SearchBox
                    v-if="state.alterType === 'MULTI_DB'"
                    v-model:value="state.searchText"
                    :placeholder="$t('common.filter-by-name')"
                  />
                </NInputGroup>
              </template>
            </NTabs>
          </template>
          <!-- standard mode project, single/multiple databases ui -->
          <template v-else>
            <div>
              <ProjectStandardView
                :state="state"
                :project="state.project"
                :database-list="selectableDatabaseList"
                :environment-list="environmentList"
                @select-database="selectDatabase"
              >
                <template #header>
                  <div class="flex items-center justify-end">
                    <NInputGroup class="py-0.5 pr-2">
                      <DatabaseLabelFilter
                        v-model:selected="state.selectedLabels"
                        :database-list="rawDatabaseList"
                      />
                      <SearchBox
                        v-model:value="state.searchText"
                        :placeholder="$t('common.filter-by-name')"
                      />
                    </NInputGroup>
                  </div>
                </template>
              </ProjectStandardView>
              <SchemalessDatabaseTable
                v-if="isEditSchema"
                mode="PROJECT"
                class="px-2"
                :database-list="schemalessDatabaseList"
              />
            </div>
          </template>
        </template>
        <template v-else>
          <div class="flex flex-col gap-y-2 px-0.5">
            <div class="w-full flex flex-row justify-between items-center">
              <div class="flex items-center space-x-3">
                <ProjectSelect
                  :project="state.project?.uid ?? String(UNKNOWN_ID)"
                  @update:project="selectProject"
                />
                <div class="px-1 space-x-2">
                  <NRadio
                    :checked="state.databaseSelectedTab === 'DATABASE'"
                    value="DATABASE"
                    name="database-tab"
                    @update:checked="state.databaseSelectedTab = 'DATABASE'"
                  >
                    {{ $t("common.database") }}
                  </NRadio>
                  <NRadio
                    :checked="state.databaseSelectedTab === 'DATABASE_GROUP'"
                    value="DATABASE_GROUP"
                    name="database-tab"
                    @update:checked="handleDatabaseGroupTabSelect"
                  >
                    <div class="flex flex-row items-center">
                      <span class="mr-1">{{ $t("database-group.self") }}</span>
                      <FeatureBadge feature="bb.feature.database-grouping" />
                    </div>
                  </NRadio>
                </div>
              </div>
              <aside class="flex justify-end">
                <NInputGroup class="py-0.5">
                  <DatabaseLabelFilter
                    v-model:selected="state.selectedLabels"
                    :database-list="rawDatabaseList"
                  />
                  <SearchBox
                    v-model:value="state.searchText"
                    :placeholder="$t('common.filter-by-name')"
                  />
                </NInputGroup>
              </aside>
            </div>
            <!-- a simple table -->
            <div v-if="state.databaseSelectedTab === 'DATABASE'">
              <DatabaseV1Table
                mode="ALL_SHORT"
                table-class="border"
                :custom-click="true"
                :database-list="selectableDatabaseList"
                :show-selection-column="true"
                @select-database="
                (db: ComposedDatabase) =>
                  toggleDatabasesSelection([db as ComposedDatabase], !isDatabaseSelected(db))
              "
              >
                <template
                  #selection-all="{ databaseList: selectedDatabaseList }"
                >
                  <input
                    v-if="selectedDatabaseList.length > 0"
                    type="checkbox"
                    class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                    v-bind="getAllSelectionState(selectedDatabaseList as ComposedDatabase[])"
                    @input="
                      toggleDatabasesSelection(
                        selectedDatabaseList as ComposedDatabase[],
                        ($event.target as HTMLInputElement).checked
                      )
                    "
                  />
                </template>
                <template #selection="{ database }">
                  <input
                    type="checkbox"
                    class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                    :checked="isDatabaseSelected(database as ComposedDatabase)"
                    @click.stop="
                      toggleDatabasesSelection(
                        [database as ComposedDatabase],
                        ($event.target as HTMLInputElement).checked
                      )
                    "
                  />
                </template>
              </DatabaseV1Table>
              <SchemalessDatabaseTable
                v-if="isEditSchema"
                mode="ALL"
                :database-list="schemalessDatabaseList"
              />
            </div>
            <div v-else-if="state.databaseSelectedTab === 'DATABASE_GROUP'">
              <SelectDatabaseGroupTable
                :database-group-list="databaseGroupList"
                :selected-database-group-name="state.selectedDatabaseGroupName"
                @update="(name) => selectDatabaseGroup(name, true)"
              />
            </div>
          </div>
        </template>
      </div>
      <div
        v-if="!ready"
        class="w-full h-[20rem] flex items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>

    <!-- Only show footer in project mode -->
    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <div>
          <div
            v-if="
              flattenSelectedDatabaseUidList.length > 0 &&
              state.alterType === 'MULTI_DB'
            "
            class="textinfolabel"
          >
            {{
              $t("database.selected-n-databases", {
                n: flattenSelectedDatabaseUidList.length,
              })
            }}
          </div>
        </div>

        <div class="flex items-center justify-end gap-x-3">
          <NButton @click.prevent="cancel">
            {{ $t("common.cancel") }}
          </NButton>
          <NTooltip
            v-if="showGenerateMultiDb"
            :disabled="flattenSelectedProjectSet.size <= 1"
          >
            <template #trigger>
              <NButton
                type="primary"
                :disabled="!allowGenerateMultiDb"
                @click.prevent="generateMultiDb"
              >
                {{ $t("common.next") }}
              </NButton>
            </template>
            <span class="w-56 text-sm">
              {{ $t("database.select-databases-from-same-project") }}
            </span>
          </NTooltip>
          <NButton
            v-if="showGenerateTenant"
            type="primary"
            :disabled="!allowGenerateTenant"
            @click.prevent="generateTenant"
          >
            {{ $t("common.next") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>

  <FeatureModal
    :open="!!featureModalContext.feature"
    :feature="featureModalContext.feature"
    @cancel="featureModalContext.feature = undefined"
  />

  <SchemaEditorModal
    v-if="state.showSchemaEditorModal"
    :database-id-list="schemaEditorContext.databaseIdList"
    :alter-type="state.alterType"
    @close="state.showSchemaEditorModal = false"
  />

  <DatabaseGroupPrevEditorModal
    v-if="state.selectedDatabaseGroup"
    :issue-type="type"
    :database-group="state.selectedDatabaseGroup"
    @close="state.selectedDatabaseGroup = undefined"
  />
</template>

<script lang="ts" setup>
import {
  NButton,
  NTabs,
  NTabPane,
  NRadio,
  NInputGroup,
  NInputGroupLabel,
} from "naive-ui";
import { computed, reactive, PropType, ref, watch } from "vue";
import { watchEffect } from "vue";
import { useRouter } from "vue-router";
import {
  hasFeature,
  useCurrentUserV1,
  useSearchDatabaseV1List,
  useDatabaseV1Store,
  useEnvironmentV1List,
  useProjectV1Store,
  useDBGroupStore,
} from "@/store";
import {
  ComposedDatabase,
  ComposedDatabaseGroup,
  FeatureType,
  UNKNOWN_ID,
  DEFAULT_PROJECT_V1_NAME,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { Project, TenantMode } from "@/types/proto/v1/project_service";
import {
  allowUsingSchemaEditorV1,
  instanceV1HasAlterSchema,
  filterDatabaseV1ByKeyword,
  sortDatabaseV1List,
  projectV1Slug,
  generateIssueName,
} from "@/utils";
import SelectDatabaseGroupTable from "../DatabaseGroup/SelectDatabaseGroupTable.vue";
import {
  DatabaseLabelFilter,
  DatabaseV1Table,
  DrawerContent,
  ProjectSelect,
} from "../v2";
import DatabaseGroupPrevEditorModal from "./DatabaseGroupPrevEditorModal.vue";
import ProjectStandardView, {
  ProjectStandardViewState,
} from "./ProjectStandardView.vue";
import ProjectTenantView, {
  ProjectTenantViewState,
} from "./ProjectTenantView.vue";
import SchemaEditorModal from "./SchemaEditorModal.vue";
import SchemalessDatabaseTable from "./SchemalessDatabaseTable.vue";

type LocalState = ProjectStandardViewState &
  ProjectTenantViewState & {
    project?: Project;
    searchText: string;
    databaseSelectedTab: "DATABASE" | "DATABASE_GROUP";
    showSchemaLessDatabaseList: boolean;
    showSchemaEditorModal: boolean;
    selectedDatabaseGroupName?: string;
    // Using to display the database group prev editor.
    selectedDatabaseGroup?: ComposedDatabaseGroup;
    selectedLabels: { key: string; value: string }[];
  };

const props = defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
  type: {
    type: String as PropType<
      "bb.issue.database.schema.update" | "bb.issue.database.data.update"
    >,
    required: true,
  },
});

const emit = defineEmits(["dismiss"]);

const router = useRouter();

const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const databaseV1Store = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();

const featureModalContext = ref<{
  feature?: FeatureType;
}>({});

const schemaEditorContext = ref<{
  databaseIdList: string[];
}>({
  databaseIdList: [],
});

const state = reactive<LocalState>({
  project: props.projectId
    ? projectV1Store.getProjectByUID(props.projectId)
    : undefined,
  alterType: "MULTI_DB",
  selectedDatabaseUidListForEnvironment: new Map<string, string[]>(),
  selectedDatabaseIdListForTenantMode: new Set<string>(),
  deployingTenantDatabaseList: [],
  label: "environment",
  searchText: "",
  databaseSelectedTab: "DATABASE",
  showSchemaLessDatabaseList: false,
  showSchemaEditorModal: false,
  selectedLabels: [],
});

const selectProject = (projectId: string | undefined) => {
  state.project = projectId
    ? projectV1Store.getProjectByUID(projectId)
    : undefined;
};

watch(
  () => props.projectId,
  (projectId) => selectProject(projectId)
);

// Returns true if alter schema, false if change data.
const isEditSchema = computed((): boolean => {
  return props.type === "bb.issue.database.schema.update";
});

const isTenantProject = computed((): boolean => {
  return (
    !!props.projectId &&
    state.project?.tenantMode === TenantMode.TENANT_MODE_ENABLED
  );
});

if (isTenantProject.value) {
  // For tenant mode projects, alter multiple db via DeploymentConfig
  // is the default suggested way.
  state.alterType = "TENANT";
}

const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const { ready } = useSearchDatabaseV1List({
  parent: "instances/-",
});

const prepareDatabaseGroupList = async () => {
  let list: ComposedDatabaseGroup[] = [];
  if (state.project) {
    list = await dbGroupStore.getOrFetchDBGroupListByProjectName(
      state.project.name
    );
  } else {
    list = await dbGroupStore.fetchAllDatabaseGroupList();
  }

  for (const group of list) {
    await dbGroupStore.getOrFetchSchemaGroupListByDBGroupName(group.name);
  }
};

watchEffect(async () => {
  await prepareDatabaseGroupList();
});

const rawDatabaseList = computed(() => {
  let list: ComposedDatabase[] = [];
  if (state.project) {
    list = databaseV1Store.databaseListByProject(state.project.name);
  } else {
    list = databaseV1Store.databaseListByUser(currentUserV1.value);
  }
  list = list.filter(
    (db) =>
      db.syncState == State.ACTIVE && db.project !== DEFAULT_PROJECT_V1_NAME
  );
  return list;
});

const filteredDatabaseList = computed(() => {
  let list = [...rawDatabaseList.value];

  list = list.filter((db) => {
    return filterDatabaseV1ByKeyword(db, state.searchText.trim(), [
      "name",
      "environment",
      "instance",
      "project",
    ]);
  });

  const labels = state.selectedLabels;
  if (labels.length > 0) {
    list = list.filter((db) => {
      return labels.some((kv) => db.labels[kv.key] === kv.value);
    });
  }

  return sortDatabaseV1List(list);
});

const selectableDatabaseList = computed(() => {
  if (isEditSchema.value) {
    return filteredDatabaseList.value.filter((db) =>
      instanceV1HasAlterSchema(db.instanceEntity)
    );
  }

  return filteredDatabaseList.value;
});

const schemalessDatabaseList = computed(() => {
  return filteredDatabaseList.value.filter(
    (db) => !instanceV1HasAlterSchema(db.instanceEntity)
  );
});

const databaseGroupList = computed(() => {
  if (state.project) {
    return dbGroupStore.getDBGroupListByProjectName(state.project.name);
  } else {
    return dbGroupStore.getAllDatabaseGroupList();
  }
});

const flattenSelectedDatabaseUidList = computed(() => {
  const flattenDatabaseIdList: string[] = [];
  if (!props.projectId) {
    for (const db of state.selectedDatabaseIdListForTenantMode) {
      flattenDatabaseIdList.push(db);
    }
  } else {
    if (isTenantProject.value && state.alterType === "MULTI_DB") {
      for (const db of state.selectedDatabaseIdListForTenantMode) {
        flattenDatabaseIdList.push(db);
      }
    } else {
      for (const databaseIdList of state.selectedDatabaseUidListForEnvironment.values()) {
        flattenDatabaseIdList.push(...databaseIdList);
      }
    }
  }
  return flattenDatabaseIdList;
});

const flattenSelectedProjectSet = computed(() => {
  const projectSet: Set<string> = new Set();
  for (const uid of flattenSelectedDatabaseUidList.value) {
    const database = databaseV1Store.getDatabaseByUID(uid);
    projectSet.add(database.projectEntity.uid);
  }
  return projectSet;
});

const showGenerateMultiDb = computed(() => {
  if (isTenantProject.value) return false;
  return state.alterType === "MULTI_DB";
});

const allowGenerateMultiDb = computed(() => {
  if (state.databaseSelectedTab === "DATABASE") {
    return (
      flattenSelectedProjectSet.value.size === 1 &&
      flattenSelectedDatabaseUidList.value.length > 0
    );
  } else {
    return state.selectedDatabaseGroupName;
  }
});

// Also works when single db selected.
const generateMultiDb = async () => {
  if (
    state.databaseSelectedTab === "DATABASE_GROUP" &&
    state.selectedDatabaseGroupName
  ) {
    const databaseGroup = await dbGroupStore.getOrFetchDBGroupByName(
      state.selectedDatabaseGroupName
    );
    state.selectedDatabaseGroup = databaseGroup;
    return;
  }

  const selectedDatabaseList = flattenSelectedDatabaseUidList.value.map(
    (id) => selectableDatabaseList.value.find((db) => db.uid === id)!
  );

  if (isEditSchema.value && allowUsingSchemaEditorV1(selectedDatabaseList)) {
    schemaEditorContext.value.databaseIdList = [
      ...flattenSelectedDatabaseUidList.value,
    ];
    state.showSchemaEditorModal = true;
    return;
  }

  if (flattenSelectedProjectSet.value.size !== 1) {
    return;
  }

  const query: Record<string, any> = {
    template: props.type,
    name: generateIssueName(
      props.type,
      selectedDatabaseList.map((db) => db.databaseName)
    ),
    project: [...flattenSelectedProjectSet.value][0],
    // The server-side will sort the databases by environment.
    // So we need not to sort them here.
    databaseList: flattenSelectedDatabaseUidList.value.join(","),
  };
  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  });
};

const showGenerateTenant = computed(() => {
  // True when a tenant project is selected and "TENANT" is selected.
  if (isTenantProject.value) {
    return true;
  }
  return false;
});

const allowGenerateTenant = computed(() => {
  if (state.databaseSelectedTab === "DATABASE") {
    if (isTenantProject.value && state.alterType === "MULTI_DB") {
      if (state.selectedDatabaseIdListForTenantMode.size === 0) {
        return false;
      }
    }

    if (isTenantProject.value) {
      // not allowed when database list filtered by deployment config is empty
      // which means no database will be deployed
      return state.deployingTenantDatabaseList.length > 0;
    }

    return true;
  } else {
    return state.selectedDatabaseGroupName;
  }
});

const getAllSelectionState = (
  databaseList: ComposedDatabase[]
): { checked: boolean; indeterminate: boolean } => {
  const set = state.selectedDatabaseIdListForTenantMode;

  const checked = set.size > 0 && databaseList.every((db) => set.has(db.uid));
  const indeterminate = !checked && databaseList.some((db) => set.has(db.uid));

  return {
    checked,
    indeterminate,
  };
};

const toggleDatabasesSelection = (
  databaseList: ComposedDatabase[],
  on: boolean
): void => {
  if (on) {
    databaseList.forEach((db) => {
      state.selectedDatabaseIdListForTenantMode.add(db.uid);
    });
  } else {
    databaseList.forEach((db) => {
      state.selectedDatabaseIdListForTenantMode.delete(db.uid);
    });
  }
};

const selectDatabaseGroup = async (
  databaseGroupName: string,
  showModal = false
) => {
  state.selectedDatabaseGroupName = databaseGroupName;

  if (showModal) {
    const databaseGroup = await dbGroupStore.getOrFetchDBGroupByName(
      state.selectedDatabaseGroupName
    );
    state.selectedDatabaseGroup = databaseGroup;
  }
};

const isDatabaseSelected = (database: ComposedDatabase): boolean => {
  return state.selectedDatabaseIdListForTenantMode.has(database.uid);
};

const handleDatabaseGroupTabSelect = () => {
  if (!hasFeature("bb.feature.database-grouping")) {
    state.databaseSelectedTab = "DATABASE";
    featureModalContext.value.feature = "bb.feature.database-grouping";
    return;
  }
  state.databaseSelectedTab = "DATABASE_GROUP";
};

const generateTenant = async () => {
  if (
    state.databaseSelectedTab === "DATABASE_GROUP" &&
    state.selectedDatabaseGroupName
  ) {
    const databaseGroup = await dbGroupStore.getOrFetchDBGroupByName(
      state.selectedDatabaseGroupName
    );
    state.selectedDatabaseGroup = databaseGroup;
    return;
  }

  if (!hasFeature("bb.feature.multi-tenancy")) {
    featureModalContext.value.feature = "bb.feature.multi-tenancy";
    return;
  }

  if (!state.project) return;
  if (state.project.uid === String(UNKNOWN_ID)) return;

  const query: Record<string, any> = {
    template: props.type,
    project: state.project.uid,
    batch: "1",
  };
  if (state.alterType === "TENANT") {
    const databaseList = databaseV1Store.databaseListByProject(
      state.project.name
    );
    if (isEditSchema.value && allowUsingSchemaEditorV1(databaseList)) {
      schemaEditorContext.value.databaseIdList = databaseList
        .filter((database) => database.syncState === State.ACTIVE)
        .map((database) => database.uid);
      state.showSchemaEditorModal = true;
      return;
    }
    // In tenant deploy pipeline, we use project name instead of database name
    // if more than one databases are to be deployed.
    const name =
      databaseList.length > 1
        ? state.project.title
        : databaseList[0].databaseName;
    query.name = generateIssueName(props.type, [name], false);
    query.databaseName = "";
  } else {
    const databaseList: ComposedDatabase[] = [];
    for (const databaseId of state.selectedDatabaseIdListForTenantMode) {
      const database = databaseV1Store.getDatabaseByUID(databaseId);
      if (database.syncState === State.ACTIVE) {
        databaseList.push(database);
      }
    }
    if (isEditSchema.value && allowUsingSchemaEditorV1(databaseList)) {
      schemaEditorContext.value.databaseIdList = Array.from(
        state.selectedDatabaseIdListForTenantMode.values()
      );
      state.showSchemaEditorModal = true;
      return;
    }

    query.name = generateIssueName(
      props.type,
      databaseList.map((database) => database.databaseName),
      false
    );
    query.databaseList = Array.from(
      state.selectedDatabaseIdListForTenantMode
    ).join(",");
  }

  emit("dismiss");

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  });
};

const selectDatabase = async (database: ComposedDatabase) => {
  if (
    isEditSchema.value &&
    database.syncState === State.ACTIVE &&
    allowUsingSchemaEditorV1([database])
  ) {
    schemaEditorContext.value.databaseIdList = [database.uid];
    state.showSchemaEditorModal = true;
    return;
  }

  emit("dismiss");

  const query: Record<string, any> = {
    template: props.type,
    name: generateIssueName(props.type, [database.databaseName]),
    project: database.projectEntity.uid,
    databaseList: database.uid,
  };
  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  });
};

const cancel = () => {
  emit("dismiss");
};
</script>
