<template>
  <DrawerContent :title="$t('quick-action.transfer-in-db-title')">
    <div class="px-4 w-[calc(100vw-8rem)] lg:w-240 max-w-[calc(100vw-8rem)]">
      <div class="flex flex-col gap-y-4">
        <TransferSourceSelector
          v-model:transfer-source="state.transferSource"
          v-model:search-text="state.searchText"
          v-model:environment="state.environmentFilter"
          v-model:instance="state.instanceFilter"
          :source-project-name="sourceProject.name"
          :has-permission-for-default-project="hasPermissionForDefaultProject"
        />
        <ProjectSelect
          v-if="state.transferSource == 'OTHER'"
          class="w-48!"
          :include-all="false"
          :include-default-project="true"
          :value="state.fromProjectName"
          :filter="filterSourceProject"
          @update:value="changeProjectFilter($event as (string | undefined))"
        />
        <template
          v-if="state.transferSource === 'OTHER' && !state.fromProjectName"
        >
          <!-- Empty -->
        </template>
        <div v-else class="w-full relative">
          <PagedDatabaseTable
            mode="PROJECT_SHORT"
            :parent="sourceProjectName"
            :filter="filter"
            :show-selection="true"
            :custom-click="true"
            v-model:selected-database-names="state.selectedDatabaseNameList"
          />
        </div>
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
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive, toRef, watchEffect } from "vue";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectByName,
} from "@/store";
import {
  type ComposedDatabase,
  DEFAULT_PROJECT_NAME,
  formatEnvironmentName,
  isValidProjectName,
} from "@/types";
import {
  BatchUpdateDatabasesRequestSchema,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Environment } from "@/types/v1/environment";
import { hasWorkspacePermissionV2 } from "@/utils";
import { DrawerContent, ProjectSelect } from "../v2";
import { PagedDatabaseTable } from "../v2/Model/DatabaseV1Table";
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

const props = withDefaults(
  defineProps<{
    projectName: string;
    onSuccess?: (databases: ComposedDatabase[]) => void;
  }>(),
  {
    onSuccess: (_: ComposedDatabase[]) => {},
  }
);

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

const sourceProjectName = computed(() => {
  if (state.transferSource === "DEFAULT") {
    return DEFAULT_PROJECT_NAME;
  }
  if (state.fromProjectName) {
    return state.fromProjectName;
  }
  return DEFAULT_PROJECT_NAME;
});

const { project: sourceProject } = useProjectByName(sourceProjectName);

const filter = computed(() => ({
  instance: state.instanceFilter?.name,
  environment: state.environmentFilter
    ? formatEnvironmentName(state.environmentFilter.id)
    : undefined,
  query: state.searchText,
}));

const allowTransfer = computed(() => state.selectedDatabaseNameList.length > 0);

const selectedDatabaseList = computed(() =>
  state.selectedDatabaseNameList.map((name) =>
    databaseStore.getDatabaseByName(name)
  )
);

const hasPermissionForDefaultProject = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.databases.list") &&
    hasWorkspacePermissionV2("bb.projects.update")
  );
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

const filterSourceProject = (project: Project) => {
  return project.name !== props.projectName;
};

const transferDatabase = async () => {
  try {
    state.loading = true;

    const updated = await useDatabaseV1Store().batchUpdateDatabases(
      create(BatchUpdateDatabasesRequestSchema, {
        parent: "-",
        requests: selectedDatabaseList.value.map((database) => {
          return create(UpdateDatabaseRequestSchema, {
            database: create(DatabaseSchema$, {
              name: database.name,
              project: props.projectName,
            }),
            updateMask: create(FieldMaskSchema, { paths: ["project"] }),
          });
        }),
      })
    );

    const displayDatabaseName =
      selectedDatabaseList.value.length > 1
        ? `${selectedDatabaseList.value.length} databases`
        : `'${selectedDatabaseList.value[0].databaseName}'`;

    props.onSuccess(updated);
    emit("dismiss");

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
