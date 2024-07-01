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
      </div>
    </template>

    <div
      class="space-y-4 h-full w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div v-if="ready">
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
      </div>
      <div
        v-if="!ready"
        class="w-full h-[20rem] flex items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>

    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <div>
          <NTooltip v-if="flattenSelectedDatabaseUidList.length > 0">
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
          <NTooltip :disabled="flattenSelectedProjectList.length <= 1">
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
    :alter-type="'MULTI_DB'"
    :plan-only="state.planOnly"
    @close="state.showSchemaEditorModal = false"
  />
</template>

<script lang="ts" setup>
import { head, uniqBy } from "lodash-es";
import { NButton, NTabs, NTabPane, NCheckbox, NTooltip } from "naive-ui";
import type { PropType } from "vue";
import { computed, reactive, ref, watchEffect, h } from "vue";
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
  useProjectV1Store,
  useDBGroupStore,
  usePageMode,
} from "@/store";
import type { ComposedDatabase, FeatureType } from "@/types";
import { UNKNOWN_ID, DEFAULT_PROJECT_V1_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
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
import SchemaEditorModal from "./SchemaEditorModal.vue";
import SchemalessDatabaseTable from "./SchemalessDatabaseTable.vue";

type LocalState = {
  label: string;
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

const { ready } = useSearchDatabaseV1List(
  computed(() => {
    const filters = ["instance = instances/-"];
    const project = selectedProject.value;
    if (project) {
      filters.push(`project = ${project.name}`);
    }
    return {
      filter: filters.join(" && "),
    };
  })
);

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

const cancel = () => {
  emit("dismiss");
};
</script>
