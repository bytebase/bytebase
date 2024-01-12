<template>
  <DrawerContent>
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>
          {{
            isEditSchema
              ? $t("database.edit-schema")
              : $t("database.change-data")
          }}
        </span>
        <i18n-t
          v-if="isTenantProject"
          class="text-sm textinfolabel"
          tag="span"
          keypath="deployment-config.pipeline-generated-from-deployment-config"
        >
          <template #deployment_config>
            <router-link
              :to="`/${selectedProject!.name}/settings`"
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
      class="space-y-4 h-full w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div v-if="ready">
        <!-- tenant mode project -->
        <template v-if="isTenantProject">
          <NTabs v-model:value="state.alterType">
            <NTabPane :tab="$t('alter-schema.alter-db-group')" name="TENANT">
              <div>
                <ProjectTenantView
                  :label="state.label"
                  :database-list="selectableDatabaseList"
                  :environment-list="environmentList"
                  :project="selectedProject"
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
                <ProjectStandardView
                  :project="selectedProject"
                  :database-list="selectableDatabaseList"
                  :environment-list="environmentList"
                  @select-databases="
                    (...dbUidList) =>
                      (state.selectedDatabaseUidList = new Set(dbUidList))
                  "
                />
                <SchemalessDatabaseTable
                  v-if="isEditSchema"
                  mode="PROJECT"
                  :database-list="schemalessDatabaseList"
                />
              </div>
              <div v-else-if="state.databaseSelectedTab === 'DATABASE_GROUP'">
                <SelectDatabaseGroupTable
                  :show-selection="true"
                  :database-group-list="filteredDatabaseGroupList"
                  :selected-database-group-name="
                    state.selectedDatabaseGroupName
                  "
                  @update="(name) => selectDatabaseGroup(name, true)"
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
                  v-if="
                    state.alterType === 'TENANT' ||
                    state.databaseSelectedTab === 'DATABASE'
                  "
                  v-model:selected="state.selectedLabels"
                  :database-list="rawDatabaseList"
                  :placement="'left-start'"
                />
                <SearchBox
                  v-if="state.alterType === 'MULTI_DB'"
                  v-model:value="state.params.query"
                  :placeholder="$t('common.filter-by-name')"
                />
              </NInputGroup>
            </template>
          </NTabs>
        </template>
        <template v-else-if="!projectId">
          <NTabs v-model:value="state.databaseSelectedTab">
            <NTabPane :tab="$t('common.database')" name="DATABASE">
              <div class="space-y-3">
                <div class="w-full flex items-center space-x-2">
                  <AdvancedSearchBox
                    v-model:params="state.params"
                    :autofocus="false"
                    :placeholder="$t('database.filter-database')"
                    :support-option-id-list="supportOptionIdList"
                  />
                  <DatabaseLabelFilter
                    v-model:selected="state.selectedLabels"
                    :database-list="rawDatabaseList"
                    :placement="'left-start'"
                  />
                </div>
                <DatabaseV1Table
                  mode="ALL_SHORT"
                  table-class="border"
                  :custom-click="true"
                  :database-list="selectableDatabaseList"
                  :show-selection-column="true"
                  :show-sql-editor-button="false"
                  :show-placeholder="true"
                  @select-database="
                (db: ComposedDatabase) =>
                  toggleDatabasesSelection([db as ComposedDatabase], !isDatabaseSelected(db))
              "
                >
                  <template
                    #selection-all="{ databaseList: selectedDatabaseList }"
                  >
                    <NCheckbox
                      v-if="selectedDatabaseList.length > 0"
                      class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                      v-bind="getAllSelectionState(selectedDatabaseList as ComposedDatabase[])"
                      @update:checked="
                        toggleDatabasesSelection(
                          selectedDatabaseList as ComposedDatabase[],
                          $event
                        )
                      "
                    />
                  </template>
                  <template #selection="{ database }">
                    <NCheckbox
                      :checked="isDatabaseSelected(database as ComposedDatabase)"
                      @update:checked="
                        toggleDatabasesSelection(
                          [database as ComposedDatabase],
                          $event
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
            </NTabPane>
            <NTabPane :tab="$t('database-group.self')" name="DATABASE_GROUP">
              <div class="space-y-3">
                <AdvancedSearchBox
                  v-model:params="state.params"
                  :autofocus="false"
                  :placeholder="$t('database.filter-database')"
                  :support-option-id-list="supportOptionIdList"
                />
                <SelectDatabaseGroupTable
                  :database-group-list="filteredDatabaseGroupList"
                  :selected-database-group-name="
                    state.selectedDatabaseGroupName
                  "
                  :show-selection="true"
                  @update="(name) => selectDatabaseGroup(name, true)"
                />
              </div>
            </NTabPane>
          </NTabs>
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
          <NButton
            v-if="isTenantProject"
            type="primary"
            :disabled="!allowGenerateTenant"
            @click.prevent="generateTenant"
          >
            {{ $t("common.next") }}
          </NButton>
          <NTooltip v-else :disabled="flattenSelectedProjectList.length <= 1">
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
import { uniqBy } from "lodash-es";
import {
  NButton,
  NTabs,
  NTabPane,
  NRadio,
  NInputGroup,
  NInputGroupLabel,
  NCheckbox,
} from "naive-ui";
import { computed, reactive, PropType, ref, watch } from "vue";
import { watchEffect } from "vue";
import { useRouter } from "vue-router";
import { PROJECT_V1_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
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
import { TenantMode } from "@/types/proto/v1/project_service";
import {
  allowUsingSchemaEditorV1,
  instanceV1HasAlterSchema,
  filterDatabaseV1ByKeyword,
  sortDatabaseV1List,
  generateIssueName,
  SearchScopeId,
  SearchParams,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";
import { DatabaseLabelFilter, DatabaseV1Table, DrawerContent } from "../v2";
import DatabaseGroupPrevEditorModal from "./DatabaseGroupPrevEditorModal.vue";
import ProjectStandardView from "./ProjectStandardView.vue";
import ProjectTenantView from "./ProjectTenantView.vue";
import SchemaEditorModal from "./SchemaEditorModal.vue";
import SchemalessDatabaseTable from "./SchemalessDatabaseTable.vue";
import SelectDatabaseGroupTable from "./SelectDatabaseGroupTable.vue";

type LocalState = {
  label: string;
  alterType: "MULTI_DB" | "TENANT";
  databaseSelectedTab: "DATABASE" | "DATABASE_GROUP";
  showSchemaLessDatabaseList: boolean;
  showSchemaEditorModal: boolean;
  selectedDatabaseGroupName?: string;
  // Using to display the database group prev editor.
  selectedDatabaseGroup?: ComposedDatabaseGroup;
  selectedLabels: { key: string; value: string }[];
  selectedDatabaseUidList: Set<string>;
  params: SearchParams;
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
  alterType: "MULTI_DB",
  selectedDatabaseUidList: new Set<string>(),
  label: "environment",
  databaseSelectedTab: "DATABASE",
  showSchemaLessDatabaseList: false,
  showSchemaEditorModal: false,
  selectedLabels: [],
  params: {
    query: "",
    scopes: [],
  },
});

const selectedProject = computed(() => {
  if (props.projectId) {
    return projectV1Store.getProjectByUID(props.projectId);
  }
  const filter = state.params.scopes.find(
    (scope) => scope.id === "project"
  )?.value;
  if (filter) {
    return projectV1Store.getProjectByName(`projects/${filter}`);
  }
  return undefined;
});

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

// Returns true if alter schema, false if change data.
const isEditSchema = computed((): boolean => {
  return props.type === "bb.issue.database.schema.update";
});

const isTenantProject = computed((): boolean => {
  return (
    !!props.projectId &&
    selectedProject.value?.tenantMode === TenantMode.TENANT_MODE_ENABLED
  );
});

watch(
  () => isTenantProject.value,
  (isTenant) => {
    if (isTenant) {
      // For tenant mode projects, alter multiple db via DeploymentConfig
      // is the default suggested way.
      state.alterType = "TENANT";
      state.databaseSelectedTab = "DATABASE_GROUP";
    }
  },
  { immediate: true }
);

const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const { ready } = useSearchDatabaseV1List({
  parent: "instances/-",
});

const prepareDatabaseGroupList = async () => {
  let list: ComposedDatabaseGroup[] = [];
  if (selectedProject.value) {
    list = await dbGroupStore.getOrFetchDBGroupListByProjectName(
      selectedProject.value.name
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
  if (selectedProject.value) {
    list = databaseV1Store.databaseListByProject(selectedProject.value.name);
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
    if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
      return (
        extractEnvironmentResourceName(db.effectiveEnvironment) ===
        selectedEnvironment.value
      );
    }
    if (selectedInstance.value !== `${UNKNOWN_ID}`) {
      return (
        extractInstanceResourceName(db.instance) === selectedInstance.value
      );
    }
    return filterDatabaseV1ByKeyword(db, state.params.query.trim(), [
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
  if (selectedProject.value) {
    return dbGroupStore.getDBGroupListByProjectName(selectedProject.value.name);
  } else {
    return dbGroupStore.getAllDatabaseGroupList();
  }
});

const filteredDatabaseGroupList = computed(() => {
  return databaseGroupList.value.filter((dbGroup) => {
    if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
      return (
        extractEnvironmentResourceName(dbGroup.environment.name) ===
        selectedEnvironment.value
      );
    }
    const keyword = state.params.query.trim().toLowerCase();
    if (!keyword) {
      return true;
    }
    return dbGroup.databaseGroupName.toLowerCase().includes(keyword);
  });
});

const flattenSelectedDatabaseUidList = computed(() => {
  return [...state.selectedDatabaseUidList];
});

const flattenSelectedProjectList = computed(() => {
  const projects = flattenSelectedDatabaseUidList.value.map((uid) => {
    return databaseV1Store.getDatabaseByUID(uid).projectEntity;
  });
  return uniqBy(projects, (project) => project.name);
});

const allowGenerateMultiDb = computed(() => {
  if (state.databaseSelectedTab === "DATABASE") {
    return (
      flattenSelectedProjectList.value.length === 1 &&
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

  if (flattenSelectedProjectList.value.length !== 1) {
    return;
  }

  const project = flattenSelectedProjectList.value[0];
  const query: Record<string, any> = {
    template: props.type,
    name: generateIssueName(
      props.type,
      selectedDatabaseList.map((db) => db.databaseName)
    ),
    project: project.uid,
    // The server-side will sort the databases by environment.
    // So we need not to sort them here.
    databaseList: flattenSelectedDatabaseUidList.value.join(","),
  };
  router.push({
    name: PROJECT_V1_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.name),
      issueSlug: "create",
    },
    query,
  });
};

const allowGenerateTenant = computed(() => {
  if (!isTenantProject.value) {
    return false;
  }
  if (state.alterType === "MULTI_DB") {
    if (state.databaseSelectedTab === "DATABASE") {
      return flattenSelectedDatabaseUidList.value.length > 0;
    }
    return !!state.selectedDatabaseGroupName;
  }

  return true;
});

const getAllSelectionState = (
  databaseList: ComposedDatabase[]
): { checked: boolean; indeterminate: boolean } => {
  const set = state.selectedDatabaseUidList;

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
      state.selectedDatabaseUidList.add(db.uid);
    });
  } else {
    databaseList.forEach((db) => {
      state.selectedDatabaseUidList.delete(db.uid);
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
  return state.selectedDatabaseUidList.has(database.uid);
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

  if (!selectedProject.value) return;
  if (selectedProject.value.uid === String(UNKNOWN_ID)) return;

  const query: Record<string, any> = {
    template: props.type,
    project: selectedProject.value.uid,
    batch: "1",
  };
  if (state.alterType === "TENANT") {
    const databaseList = databaseV1Store.databaseListByProject(
      selectedProject.value.name
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
        ? selectedProject.value.title
        : databaseList[0].databaseName;
    query.name = generateIssueName(props.type, [name], false);
    query.databaseName = "";
  } else {
    const databaseList: ComposedDatabase[] = [];
    for (const databaseId of flattenSelectedDatabaseUidList.value) {
      const database = databaseV1Store.getDatabaseByUID(databaseId);
      if (database.syncState === State.ACTIVE) {
        databaseList.push(database);
      }
    }
    if (isEditSchema.value && allowUsingSchemaEditorV1(databaseList)) {
      schemaEditorContext.value.databaseIdList = [
        ...flattenSelectedDatabaseUidList.value,
      ];
      state.showSchemaEditorModal = true;
      return;
    }

    query.name = generateIssueName(
      props.type,
      databaseList.map((database) => database.databaseName),
      false
    );
    query.databaseList = flattenSelectedDatabaseUidList.value.join(",");
  }

  emit("dismiss");

  router.push({
    name: PROJECT_V1_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(selectedProject.value.name),
      issueSlug: "create",
    },
    query,
  });
};

const cancel = () => {
  emit("dismiss");
};

const supportOptionIdList = computed((): SearchScopeId[] => {
  if (state.databaseSelectedTab === "DATABASE") {
    return ["project", "instance", "environment"];
  }

  return ["project", "environment"];
});
</script>
