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
      <div>
        <div class="space-y-3">
          <div class="w-full flex items-center space-x-2">
            <AdvancedSearch
              v-model:params="state.params"
              :autofocus="false"
              :placeholder="$t('database.filter-database')"
              :scope-options="scopeOptions"
            />
          </div>

          <PagedDatabaseTable
            mode="ALL_SHORT"
            :filter="filter"
            :parent="projectName"
            :custom-click="true"
            :show-sql-editor-button="false"
            @update:selected-databases="handleDatabasesSelectionChanged"
          />
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <div>
          <NTooltip v-if="flattenSelectedDatabaseNameList.length > 0">
            <template #trigger>
              <div class="textinfolabel">
                {{
                  $t("database.selected-n-databases", {
                    n: flattenSelectedDatabaseNameList.length,
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
    :database-names="schemaEditorContext.databaseNameList"
    :alter-type="'MULTI_DB'"
    :plan-only="state.planOnly"
    @close="state.showSchemaEditorModal = false"
  />
</template>

<script lang="ts" setup>
import { uniqBy } from "lodash-es";
import { NButton, NCheckbox, NTooltip } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { FeatureModal } from "@/components/FeatureGuard";
import { PagedDatabaseTable } from "@/components/v2/Model/DatabaseV1Table";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_REVIEW_CENTER_DETAIL,
} from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useAppFeature } from "@/store";
import {
  instanceNamePrefix,
  environmentNamePrefix,
} from "@/store/modules/v1/common";
import type { FeatureType } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { SearchParams, SearchScope } from "@/utils";
import {
  allowUsingSchemaEditor,
  generateIssueTitle,
  extractProjectResourceName,
} from "@/utils";
import AdvancedSearch from "../AdvancedSearch";
import { useCommonSearchScopeOptions } from "../AdvancedSearch/useCommonSearchScopeOptions";
import { DrawerContent } from "../v2";
import SchemaEditorModal from "./SchemaEditorModal.vue";

type LocalState = {
  showSchemaLessDatabaseList: boolean;
  selectedDatabaseNames: Set<string>;
  showSchemaEditorModal: boolean;
  params: SearchParams;
  // planOnly is used to indicate whether only to create plan.
  planOnly: boolean;
};

const props = withDefaults(
  defineProps<{
    projectName: string;
    type: "bb.issue.database.schema.update" | "bb.issue.database.data.update";
    planOnly?: boolean;
  }>(),
  {
    planOnly: false,
  }
);

const emit = defineEmits(["dismiss"]);

const router = useRouter();
const databaseV1Store = useDatabaseV1Store();
const disableSchemaEditor = useAppFeature(
  "bb.feature.issue.disable-schema-editor"
);

const featureModalContext = ref<{
  feature?: FeatureType;
}>({});

const schemaEditorContext = ref<{
  databaseNameList: string[];
}>({
  databaseNameList: [],
});

const readonlyScopes = computed((): SearchScope[] => [
  {
    id: "project",
    value: extractProjectResourceName(props.projectName),
    readonly: true,
  },
]);

const state = reactive<LocalState>({
  selectedDatabaseNames: new Set<string>(),
  showSchemaLessDatabaseList: false,
  showSchemaEditorModal: false,
  params: {
    query: "",
    scopes: [...readonlyScopes.value],
  },
  planOnly: props.planOnly,
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => ["project", "instance", "environment", "database-label"])
);

const selectedInstance = computed(() => {
  const instanceId = state.params.scopes.find(
    (scope) => scope.id === "instance"
  )?.value;
  if (!instanceId) {
    return;
  }
  return `${instanceNamePrefix}${instanceId}`;
});

const selectedEnvironment = computed(() => {
  const environmentId = state.params.scopes.find(
    (scope) => scope.id === "environment"
  )?.value;
  if (!environmentId) {
    return;
  }
  return `${environmentNamePrefix}${environmentId}`;
});

const selectedLabels = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "database-label")
    .map((scope) => scope.value);
});

const filter = computed(() => ({
  instance: selectedInstance.value,
  environment: selectedEnvironment.value,
  query: state.params.query,
  excludeEngines: [Engine.REDIS],
  labels: selectedLabels.value,
}));

// Returns true if alter schema, false if change data.
const isEditSchema = computed((): boolean => {
  return props.type === "bb.issue.database.schema.update";
});

const flattenSelectedDatabaseNameList = computed(() => {
  return [...state.selectedDatabaseNames];
});

const flattenSelectedProjectList = computed(() => {
  const projects = flattenSelectedDatabaseNameList.value.map((name) => {
    return databaseV1Store.getDatabaseByName(name).projectEntity;
  });
  return uniqBy(projects, (project) => project.name);
});

const allowGenerateMultiDb = computed(() => {
  return (
    flattenSelectedProjectList.value.length === 1 &&
    flattenSelectedDatabaseNameList.value.length > 0
  );
});

const flattenSelectedDatabaseList = computed(() =>
  flattenSelectedDatabaseNameList.value.map((name) =>
    databaseV1Store.getDatabaseByName(name)
  )
);

// Also works when single db selected.
const generateMultiDb = async () => {
  if (
    flattenSelectedDatabaseList.value.length === 1 &&
    isEditSchema.value &&
    allowUsingSchemaEditor(flattenSelectedDatabaseList.value) &&
    !disableSchemaEditor.value
  ) {
    schemaEditorContext.value.databaseNameList = [
      ...flattenSelectedDatabaseNameList.value,
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
    name: generateIssueTitle(
      props.type,
      flattenSelectedDatabaseList.value.map((db) => db.databaseName)
    ),
    databaseList: flattenSelectedDatabaseList.value
      .map((db) => db.name)
      .join(","),
  };

  router.push({
    name: state.planOnly
      ? PROJECT_V1_ROUTE_REVIEW_CENTER_DETAIL
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
  state.selectedDatabaseNames = selectedDatabaseNameList;
};

const cancel = () => {
  emit("dismiss");
};
</script>
