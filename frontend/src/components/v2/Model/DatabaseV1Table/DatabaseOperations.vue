<template>
  <NScrollbar x-scrollable>
    <div
      v-bind="$attrs"
      class="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4"
    >
      <span class="whitespace-nowrap">{{
        $t("database.selected-n-databases", {
          n: databases.length,
        })
      }}</span>
      <div class="flex items-center">
        <template v-for="action in actions" :key="action.text">
          <PermissionGuardWrapper
            v-slot="slotProps"
            :project="selectedProject"
            :permissions="action.requiredPermissions"
          >
            <NTooltip
              :disabled="slotProps.disabled || !action.disabled || !getDisabledTooltip(action.text)"
            >
              <template #trigger>
                <NButton
                  quaternary
                  size="small"
                  type="primary"
                  :disabled="slotProps.disabled || action.disabled"
                  @click="action.click"
                >
                  <template #icon>
                    <component :is="action.icon" class="h-4 w-4" />
                  </template>
                  <span class="text-sm">{{ action.text }}</span>
                </NButton>
              </template>
              <span class="w-56 text-sm">
                {{ getDisabledTooltip(action.text) }}
              </span>
            </NTooltip>
          </PermissionGuardWrapper>
        </template>
      </div>
    </div>
  </NScrollbar>

  <LabelEditorDrawer
    :show="state.showLabelEditorDrawer"
    :readonly="false"
    :title="
      $t('db.labels-for-resource', {
        resource: $t('database.n-databases', { n: databases.length }),
      })
    "
    :labels="databases.map((db) => db.labels)"
    @dismiss="state.showLabelEditorDrawer = false"
    @apply="onLabelsApply($event)"
  />

  <EditEnvironmentDrawer
    :show="state.showEditEnvironmentDrawer"
    @dismiss="state.showEditEnvironmentDrawer = false"
    @update="onEnvironmentUpdate($event)"
  />

  <Drawer
    :show="!!state.transferOutDatabaseType"
    :auto-focus="true"
    @close="state.transferOutDatabaseType = undefined"
  >
    <TransferOutDatabaseForm
      v-if="state.transferOutDatabaseType === 'TRANSFER-OUT'"
      :database-list="props.databases"
      :selected-database-names="selectedDatabaseNameList"
      :on-success="() => $emit('refresh')"
      @dismiss="state.transferOutDatabaseType = undefined"
    />
    <TransferDatabaseForm
      v-else
      :project-name="projectName"
      :on-success="() => $emit('refresh')"
      @dismiss="state.transferOutDatabaseType = undefined"
    />
  </Drawer>

  <BBAlert
    v-model:show="state.showUnassignAlert"
    type="warning"
    :ok-text="$t('common.confirm')"
    :title="$t('database.unassign-alert-title')"
    :description="$t('database.unassign-alert-description')"
    @ok="
      () => {
        state.showUnassignAlert = false;
        unAssignDatabases();
      }
    "
    @cancel="state.showUnassignAlert = false"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema as FieldMaskProtoEsSchema } from "@bufbuild/protobuf/wkt";
import {
  ArrowRightLeftIcon,
  ChevronsDownIcon,
  DownloadIcon,
  PencilIcon,
  RefreshCcwIcon,
  SquareStackIcon,
  TagIcon,
  UnlinkIcon,
} from "lucide-vue-next";
import { NButton, NScrollbar, NTooltip, useDialog } from "naive-ui";
import type { VNode } from "vue";
import { computed, h, reactive } from "vue";
import { useI18n } from "vue-i18n";
import type { LocationQueryRaw } from "vue-router";
import { useRouter } from "vue-router";
import { BBAlert } from "@/bbkit";
import EditEnvironmentDrawer from "@/components/EditEnvironmentDrawer.vue";
import LabelEditorDrawer from "@/components/LabelEditorDrawer.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { preCreateIssue } from "@/components/Plan/logic/issue";
import { TransferDatabaseForm } from "@/components/TransferDatabaseForm";
import TransferOutDatabaseForm from "@/components/TransferOutDatabaseForm";
import { Drawer } from "@/components/v2";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useGracefulRequest,
  useProjectV1Store,
} from "@/store";
import type { ComposedDatabase, Permission } from "@/types";
import { DEFAULT_PROJECT_NAME, isValidProjectName } from "@/types";
import {
  BatchUpdateDatabasesRequestSchema,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { extractProjectResourceName, generateIssueTitle } from "@/utils";

interface DatabaseAction {
  icon: VNode;
  text: string;
  disabled: boolean;
  click: () => void;
  requiredPermissions: Permission[];
}

interface LocalState {
  loading: boolean;
  showUnassignAlert: boolean;
  showLabelEditorDrawer: boolean;
  showEditEnvironmentDrawer: boolean;
  transferOutDatabaseType?: "TRANSFER-IN" | "TRANSFER-OUT";
}

const props = withDefaults(
  defineProps<{
    databases: ComposedDatabase[];
    projectName?: string;
  }>(),
  { projectName: "" }
);

const state = reactive<LocalState>({
  loading: false,
  showUnassignAlert: false,
  showLabelEditorDrawer: false,
  showEditEnvironmentDrawer: false,
  transferOutDatabaseType: undefined,
});

const emit = defineEmits<{
  (event: "refresh"): void;
  (event: "update", databases: ComposedDatabase[]): void;
}>();

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const projectStore = useProjectV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const dialog = useDialog();

const selectedProjectNames = computed(() => {
  return new Set(props.databases.map((db) => db.project));
});

const assignedDatabases = computed(() => {
  return props.databases.filter((db) => db.project !== DEFAULT_PROJECT_NAME);
});

const getDisabledTooltip = (action: string) => {
  if (selectedProjectNames.value.size > 1) {
    return t("database.batch-action-disabled", {
      action: action.toLowerCase(),
    });
  }
  if (selectedProjectNames.value.has(DEFAULT_PROJECT_NAME)) {
    return t("database.batch-action-disabled-for-unassigned", {
      action: action.toLowerCase(),
    });
  }
  return "";
};

const selectedProjectName = computed(() => {
  if (props.projectName) {
    return props.projectName;
  }
  if (selectedProjectNames.value.size !== 1) {
    return "";
  }
  return [...selectedProjectNames.value][0];
});

const selectedProject = computed(() => {
  if (!isValidProjectName(selectedProjectName.value)) {
    return;
  }
  return projectStore.getProjectByName(selectedProjectName.value);
});

const selectedDatabaseNameList = computed(() => {
  return props.databases.map((db) => db.name);
});

const operations = computed(() => {
  return [
    "CHANGE-DATABASE",
    "EXPORT-DATA",
    "SYNC-SCHEMA",
    "EDIT-LABELS",
    "EDIT-ENVIRONMENT",
    "TRANSFER-OUT",
    "TRANSFER-IN",
  ];
});

const showDatabaseDriftedWarningDialog = () => {
  return new Promise((resolve) => {
    dialog.create({
      type: "warning",
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      title: t("issue.schema-drift-detected.self"),
      content: t("issue.schema-drift-detected.description"),
      autoFocus: false,
      onNegativeClick: () => {
        resolve(false);
      },
      onPositiveClick: () => {
        resolve(true);
      },
    });
  });
};

const generateMultiDb = async (
  type: "bb.issue.database.update" | "bb.issue.database.data.export"
) => {
  // Check if any database is drifted.
  if (props.databases.some((d) => d.drifted)) {
    const confirmed = await showDatabaseDriftedWarningDialog();
    if (!confirmed) {
      return;
    }
  }

  // Fetch project to check enforce_issue_title setting
  const project = await projectStore.getOrFetchProjectByName(
    selectedProjectName.value
  );

  const { enabledNewLayout } = useIssueLayoutVersion();
  const query: LocationQueryRaw = {
    template: type,
    databaseList: props.databases.map((db) => db.name).join(","),
  };
  // Only set title from generated if enforceIssueTitle is false.
  if (!project.enforceIssueTitle) {
    query.name = generateIssueTitle(
      type,
      props.databases.map((db) => db.databaseName)
    );
  }

  const isDataExport = type === "bb.issue.database.data.export";
  if (isDataExport || enabledNewLayout.value) {
    // Use CI/CD UI for data export issues or when new layout is enabled
    router.push({
      name: isDataExport
        ? PROJECT_V1_ROUTE_PLAN_DETAIL
        : PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      params: {
        projectId: extractProjectResourceName(selectedProjectName.value),
        planId: "create",
        ...(isDataExport ? {} : { specId: "placeholder" }),
      },
      query,
    });
  } else {
    // Legacy UI for database update when new layout is disabled
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(selectedProjectName.value),
        issueSlug: "create",
      },
      query,
    });
  }
};

const syncSchema = async () => {
  if (state.loading) {
    return;
  }
  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("db.start-to-sync-schema"),
  });
  try {
    state.loading = true;
    await useGracefulRequest(async () => {
      await databaseStore.batchSyncDatabases(
        props.databases.map((db) => db.name)
      );
      for (const db of props.databases) {
        dbSchemaStore.removeCache(db.name);
      }
    });
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("db.successfully-synced-schema"),
    });
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("db.failed-to-sync-schema"),
    });
  } finally {
    state.loading = false;
  }
};

const unAssignDatabases = async () => {
  if (state.loading) {
    return;
  }
  try {
    state.loading = true;
    await databaseStore.batchUpdateDatabases(
      create(BatchUpdateDatabasesRequestSchema, {
        parent: "-",
        requests: assignedDatabases.value.map((database) => {
          return create(UpdateDatabaseRequestSchema, {
            database: create(DatabaseSchema$, {
              name: database.name,
              project: DEFAULT_PROJECT_NAME,
            }),
            updateMask: create(FieldMaskProtoEsSchema, { paths: ["project"] }),
          });
        }),
      })
    );
    emit("refresh");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("database.successfully-transferred-databases"),
    });
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("db.failed-to-sync-schema"),
    });
  } finally {
    state.loading = false;
  }
};

const operationsInProjectDetail = computed(() => !!props.projectName);

const isInDefaultProject = computed(
  () => props.projectName === DEFAULT_PROJECT_NAME
);

const actions = computed((): DatabaseAction[] => {
  const resp: DatabaseAction[] = [];
  for (const operation of operations.value) {
    switch (operation) {
      case "CHANGE-DATABASE":
        if (operationsInProjectDetail.value && !isInDefaultProject.value) {
          resp.push({
            icon: h(PencilIcon),
            text: t("database.change-database"),
            disabled:
              !selectedProjectName.value ||
              selectedDatabaseNameList.value.length < 1 ||
              selectedProjectNames.value.has(DEFAULT_PROJECT_NAME),
            click: () => {
              preCreateIssue(
                selectedProjectName.value,
                selectedDatabaseNameList.value
              );
            },
            requiredPermissions: [
              "bb.issues.create",
              "bb.plans.create",
              "bb.rollouts.create",
            ],
          });
        }
        break;
      case "EXPORT-DATA":
        if (operationsInProjectDetail.value && !isInDefaultProject.value) {
          resp.push({
            icon: h(DownloadIcon),
            text: t("custom-approval.risk-rule.risk.namespace.data_export"),
            disabled:
              !selectedProjectName.value ||
              props.databases.length < 1 ||
              selectedProjectNames.value.has(DEFAULT_PROJECT_NAME),
            click: () => generateMultiDb("bb.issue.database.data.export"),
            requiredPermissions: [
              "bb.issues.create",
              "bb.plans.create",
              "bb.rollouts.create",
            ],
          });
        }
        break;
      case "SYNC-SCHEMA":
        resp.push({
          icon: h(RefreshCcwIcon),
          text: t("database.sync-schema-button"),
          disabled: props.databases.length < 1,
          click: syncSchema,
          requiredPermissions: ["bb.databases.sync"],
        });
        break;
      case "EDIT-LABELS":
        resp.push({
          icon: h(TagIcon),
          text: t("database.edit-labels"),
          disabled: props.databases.length < 1,
          click: () => (state.showLabelEditorDrawer = true),
          requiredPermissions: ["bb.databases.update"],
        });
        break;
      case "EDIT-ENVIRONMENT":
        resp.push({
          icon: h(SquareStackIcon),
          text: t("database.edit-environment"),
          disabled: props.databases.length < 1,
          click: () => (state.showEditEnvironmentDrawer = true),
          requiredPermissions: ["bb.databases.update"],
        });
        break;
      case "TRANSFER-IN": {
        if (operationsInProjectDetail.value && !isInDefaultProject.value) {
          resp.push({
            icon: h(ChevronsDownIcon),
            text: t("quick-action.transfer-in-db"),
            disabled: false,
            click: () => (state.transferOutDatabaseType = "TRANSFER-IN"),
            requiredPermissions: ["bb.projects.update"],
          });
        }
        break;
      }
      case "TRANSFER-OUT":
        if (!operationsInProjectDetail.value) {
          resp.push({
            icon: h(ArrowRightLeftIcon),
            text: t("database.transfer-project"),
            disabled: props.databases.length < 1,
            click: () => (state.transferOutDatabaseType = "TRANSFER-OUT"),
            requiredPermissions: [],
          });
        } else if (!isInDefaultProject.value) {
          resp.push({
            icon: h(UnlinkIcon),
            text: t("database.unassign"),
            disabled: props.databases.length < 1,
            click: () => (state.showUnassignAlert = true),
            requiredPermissions: ["bb.databases.update"],
          });
        }
        break;
    }
  }

  return resp;
});

const onLabelsApply = async (labelsList: { [key: string]: string }[]) => {
  if (labelsList.length !== props.databases.length) {
    // This should never happen.
    return;
  }

  const updatedDatabases = await databaseStore.batchUpdateDatabases(
    create(BatchUpdateDatabasesRequestSchema, {
      parent: "-",
      requests: props.databases.map((database, i) => {
        const label = labelsList[i];
        const patch = create(DatabaseSchema$, {
          ...database,
          labels: label,
        });
        return create(UpdateDatabaseRequestSchema, {
          database: patch,
          updateMask: create(FieldMaskProtoEsSchema, { paths: ["labels"] }),
        });
      }),
    })
  );
  emit("update", updatedDatabases);

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const onEnvironmentUpdate = async (environment: string) => {
  const updatedDatabases = await databaseStore.batchUpdateDatabases(
    create(BatchUpdateDatabasesRequestSchema, {
      parent: "-",
      requests: props.databases.map((database) => {
        return create(UpdateDatabaseRequestSchema, {
          database: create(DatabaseSchema$, {
            name: database.name,
            environment: environment,
          }),
          updateMask: create(FieldMaskProtoEsSchema, {
            paths: ["environment"],
          }),
        });
      }),
    })
  );
  emit("update", updatedDatabases);

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
