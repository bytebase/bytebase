<template>
  <DrawerContent class="max-w-[100vw]">
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
                  @update:checked="state.databaseSelectedTab = 'DATABASE_GROUP'"
                >
                  <renderDatabaseGroupTabTitle />
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
                <DatabaseGroupDataTable
                  :database-group-list="filteredDatabaseGroupList"
                  :show-edit="false"
                  @update:selected-database-groups="
                    handleDatabaseGroupsSelectionChanged
                  "
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
        <template v-else>
          <NTabs v-model:value="state.databaseSelectedTab">
            <NTabPane :tab="$t('common.database')" name="DATABASE">
              <div class="space-y-3">
                <div class="w-full flex items-center space-x-2">
                  <AdvancedSearch
                    v-model:params="state.params"
                    :autofocus="false"
                    :placeholder="$t('database.filter-database')"
                    :scope-options="scopeOptions"
                  />
                  <DatabaseLabelFilter
                    v-model:selected="state.selectedLabels"
                    :database-list="rawDatabaseList"
                    :placement="'left-start'"
                  />
                </div>
                <DatabaseV1Table
                  mode="ALL_SHORT"
                  :show-sql-editor-button="false"
                  :database-list="selectableDatabaseList"
                  @update:selected-databases="handleDatabasesSelectionChanged"
                />
                <SchemalessDatabaseTable
                  v-if="isEditSchema"
                  mode="ALL"
                  :database-list="schemalessDatabaseList"
                />
              </div>
            </NTabPane>
            <NTabPane :tab="renderDatabaseGroupTabTitle" name="DATABASE_GROUP">
              <div class="space-y-3">
                <AdvancedSearch
                  v-model:params="state.params"
                  :autofocus="false"
                  :placeholder="$t('database.filter-database-group')"
                />
                <DatabaseGroupDataTable
                  :database-group-list="filteredDatabaseGroupList"
                  :show-edit="false"
                  @update:selected-database-groups="
                    handleDatabaseGroupsSelectionChanged
                  "
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
          <NTooltip
            v-if="
              flattenSelectedDatabaseUidList.length > 0 &&
              state.alterType === 'MULTI_DB'
            "
          >
            <template #trigger>
              <div class="textinfolabel">
                {{
                  $t("database.selected-n-databases", {
                    n: flattenSelectedDatabaseUidList.length,
                  })
                }}
              </div>
            </template>
            <div class="mx-2">
              <ul class="list-disc">
                <li v-for="db in flattenSelectedDatabaseList" :key="db.name">
                  {{ db.databaseName }}
                </li>
              </ul>
            </div>
          </NTooltip>
        </div>

        <div class="flex items-center justify-end gap-x-3">
          <NCheckbox v-if="!props.planOnly" v-model:checked="state.planOnly">
            {{ $t("issue.sql-review-only") }}
          </NCheckbox>
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
    :plan-only="state.planOnly"
    @close="state.showSchemaEditorModal = false"
  />
</template>

<script lang="ts" setup>
import { head, uniqBy } from "lodash-es";
import {
  NButton,
  NTabs,
  NTabPane,
  NRadio,
  NInputGroup,
  NInputGroupLabel,
  NCheckbox,
} from "naive-ui";
import type { PropType } from "vue";
import { computed, reactive, ref, watch, watchEffect, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { DatabaseGroupDataTable } from "@/components/DatabaseGroup";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import DatabaseV1Table from "@/components/v2/Model/DatabaseV1Table";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  hasFeature,
  useCurrentUserV1,
  useSearchDatabaseV1List,
  useDatabaseV1Store,
  useEnvironmentV1List,
  useProjectV1Store,
  useDBGroupStore,
  usePageMode,
} from "@/store";
import type { ComposedDatabase, FeatureType } from "@/types";
import { UNKNOWN_ID, DEFAULT_PROJECT_V1_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import { TenantMode } from "@/types/proto/v1/project_service";
import type { SearchParams } from "@/utils";
import {
  allowUsingSchemaEditor,
  instanceV1HasAlterSchema,
  filterDatabaseV1ByKeyword,
  sortDatabaseV1List,
  generateIssueName,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";
import { generateDatabaseGroupIssueRoute } from "@/utils/databaseGroup/issue";
import AdvancedSearch from "../AdvancedSearch";
import { useCommonSearchScopeOptions } from "../AdvancedSearch/useCommonSearchScopeOptions";
import { DatabaseLabelFilter, DrawerContent } from "../v2";
import ProjectStandardView from "./ProjectStandardView.vue";
import ProjectTenantView from "./ProjectTenantView.vue";
import SchemaEditorModal from "./SchemaEditorModal.vue";
import SchemalessDatabaseTable from "./SchemalessDatabaseTable.vue";

type LocalState = {
  label: string;
  alterType: "MULTI_DB" | "TENANT";
  databaseSelectedTab: "DATABASE" | "DATABASE_GROUP";
  showSchemaLessDatabaseList: boolean;
  selectedLabels: { key: string; value: string }[];
  selectedDatabaseUidList: Set<string>;
  showSchemaEditorModal: boolean;
  selectedDatabaseGroupName?: string;
  params: SearchParams;
  // planOnly is used to indicate whether only to create plan.
  planOnly: boolean;
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
  planOnly: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["dismiss"]);

const router = useRouter();
const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const databaseV1Store = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const pageMode = usePageMode();

const isStandaloneMode = computed(() => pageMode.value === "STANDALONE");

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
  planOnly: props.planOnly,
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  computed(() =>
    state.databaseSelectedTab === "DATABASE"
      ? ["project", "instance", "environment"]
      : ["project", "environment"]
  )
);

const hasDatabaseGroupFeature = computed(() => {
  return hasFeature("bb.feature.database-grouping");
});

const renderDatabaseGroupTabTitle = () => {
  return h("div", { class: "flex flex-row items-center space-x-1" }, [
    h("span", t("database-group.self")),
    h(FeatureBadge, { feature: "bb.feature.database-grouping" }),
  ]);
};

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
  filter: "instance = instances/-",
});

const prepareDatabaseGroupList = async () => {
  if (selectedProject.value) {
    await dbGroupStore.getOrFetchDBGroupListByProjectName(
      selectedProject.value.name
    );
  } else {
    await dbGroupStore.fetchAllDatabaseGroupList();
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
  const list = rawDatabaseList.value.filter((db) => {
    if (
      selectedEnvironment.value !== `${UNKNOWN_ID}` &&
      extractEnvironmentResourceName(db.effectiveEnvironment) !==
        selectedEnvironment.value
    ) {
      return false;
    }
    if (
      selectedInstance.value !== `${UNKNOWN_ID}` &&
      extractInstanceResourceName(db.instance) !== selectedInstance.value
    ) {
      return false;
    }

    const filterByKeyword = filterDatabaseV1ByKeyword(
      db,
      state.params.query.trim(),
      ["name", "environment", "instance", "project"]
    );
    if (!filterByKeyword) {
      return false;
    }

    if (state.selectedLabels.length > 0) {
      return state.selectedLabels.some((kv) => db.labels[kv.key] === kv.value);
    }

    return true;
  });

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

const previewDatabaseGroupIssue = () => {
  if (!state.selectedDatabaseGroupName) {
    // Should not reach here.
    return;
  }

  const databaseGroup = dbGroupStore.getDBGroupByName(
    state.selectedDatabaseGroupName
  );
  if (!databaseGroup) {
    console.error("Database group not found");
    return;
  }

  router.push(
    generateDatabaseGroupIssueRoute(
      props.type,
      databaseGroup,
      "",
      state.planOnly
    )
  );
};

const flattenSelectedDatabaseList = computed(() =>
  flattenSelectedDatabaseUidList.value.map(
    (id) => selectableDatabaseList.value.find((db) => db.uid === id)!
  )
);

// Also works when single db selected.
const generateMultiDb = async () => {
  if (
    state.databaseSelectedTab === "DATABASE_GROUP" &&
    state.selectedDatabaseGroupName
  ) {
    previewDatabaseGroupIssue();
    return;
  }

  if (
    flattenSelectedDatabaseList.value.length === 1 &&
    isEditSchema.value &&
    allowUsingSchemaEditor(flattenSelectedDatabaseList.value) &&
    !isStandaloneMode.value
  ) {
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
      flattenSelectedDatabaseList.value.map((db) => db.databaseName)
    ),
    databaseList: flattenSelectedDatabaseList.value
      .map((db) => db.name)
      .join(","),
  };

  router.push({
    name: state.planOnly
      ? PROJECT_V1_ROUTE_PLAN_DETAIL
      : PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.name),
      issueSlug: "create",
      planSlug: "create",
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

const handleDatabasesSelectionChanged = (
  selectedDatabaseNameList: Set<string>
) => {
  state.selectedDatabaseUidList = new Set(
    Array.from(selectedDatabaseNameList).map(
      (name) => databaseV1Store.getDatabaseByName(name)?.uid
    )
  );
};

const handleDatabaseGroupsSelectionChanged = (
  databaseGroupNames: Set<string>
): void => {
  const databaseGroupName = head(Array.from(databaseGroupNames));
  if (!databaseGroupName) return;
  if (!hasDatabaseGroupFeature.value) {
    featureModalContext.value.feature = "bb.feature.database-grouping";
    return;
  }
  state.selectedDatabaseGroupName = databaseGroupName;
};

const generateTenant = async () => {
  if (
    state.databaseSelectedTab === "DATABASE_GROUP" &&
    state.selectedDatabaseGroupName
  ) {
    previewDatabaseGroupIssue();
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
    batch: "1",
  };
  if (state.alterType === "TENANT") {
    const databaseList = databaseV1Store.databaseListByProject(
      selectedProject.value.name
    );
    if (isEditSchema.value && allowUsingSchemaEditor(databaseList)) {
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
    if (isEditSchema.value && allowUsingSchemaEditor(databaseList)) {
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
    query.databaseList = databaseList.map((db) => db.name).join(",");
  }

  emit("dismiss");

  router.push({
    name: state.planOnly
      ? PROJECT_V1_ROUTE_PLAN_DETAIL
      : PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(selectedProject.value.name),
      issueSlug: "create",
      planSlug: "create",
    },
    query,
  });
};

const cancel = () => {
  emit("dismiss");
};
</script>
