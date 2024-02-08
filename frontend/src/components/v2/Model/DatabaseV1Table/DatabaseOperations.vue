<template>
  <div
    v-bind="$attrs"
    class="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4 overflow-x-auto"
  >
    {{
      $t("database.selected-n-databases", {
        n: databases.length,
      })
    }}
    <div class="flex items-center">
      <template v-for="action in actions" :key="action.text">
        <NTooltip :disabled="!action.disabled || !action.tooltip(action.text)">
          <template #trigger>
            <NButton
              quaternary
              size="small"
              type="primary"
              :disabled="action.disabled"
              @click="action.click"
            >
              <template #icon>
                <component :is="action.icon" class="h-4 w-4" />
              </template>
              <span class="text-sm">{{ action.text }}</span>
            </NButton>
          </template>
          <span class="w-56 text-sm">
            {{ action.tooltip(action.text.toLowerCase()) }}
          </span>
        </NTooltip>
      </template>
    </div>
  </div>

  <SchemaEditorModal
    v-if="state.showSchemaEditorModal"
    :database-id-list="schemaEditorContext.databaseIdList"
    :alter-type="'MULTI_DB'"
    @close="state.showSchemaEditorModal = false"
  />

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

  <Drawer
    :show="state.showTransferOutDatabaseForm"
    :auto-focus="true"
    @close="state.showTransferOutDatabaseForm = false"
  >
    <TransferOutDatabaseForm
      :database-list="props.databases"
      :selected-database-uid-list="selectedDatabaseUidList"
      @dismiss="state.showTransferOutDatabaseForm = false"
    />
  </Drawer>

  <BBAlert
    v-model:show="state.showUnassignAlert"
    type="warning"
    :ok-text="$t('database.unassign')"
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
import {
  UnlinkIcon,
  RefreshCcwIcon,
  TagIcon,
  PencilIcon,
  PenSquareIcon,
  ArrowRightLeftIcon,
} from "lucide-vue-next";
import { computed, h, VNode, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { Drawer } from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useCurrentUserV1,
  useCurrentUserIamPolicy,
  useProjectV1Store,
  useDatabaseV1Store,
  useGracefulRequest,
  useDBSchemaV1Store,
  pushNotification,
  usePageMode,
} from "@/store";
import {
  ComposedDatabase,
  DEFAULT_PROJECT_V1_NAME,
  ProjectPermission,
} from "@/types";
import {
  Database,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import {
  isArchivedDatabaseV1,
  instanceV1HasAlterSchema,
  allowUsingSchemaEditorV1,
  generateIssueName,
  hasProjectPermissionV2,
  extractProjectResourceName,
} from "@/utils";

interface DatabaseAction {
  icon: VNode;
  text: string;
  disabled: boolean;
  click: () => void;
  tooltip: (action: string) => string;
}

interface LocalState {
  loading: boolean;
  showSchemaEditorModal: boolean;
  showUnassignAlert: boolean;
  showLabelEditorDrawer: boolean;
  showTransferOutDatabaseForm: boolean;
}

const props = withDefaults(
  defineProps<{
    databases: ComposedDatabase[];
    projectUid?: string;
  }>(),
  { projectUid: "" }
);

const state = reactive<LocalState>({
  loading: false,
  showSchemaEditorModal: false,
  showUnassignAlert: false,
  showLabelEditorDrawer: false,
  showTransferOutDatabaseForm: false,
});
const schemaEditorContext = ref<{
  databaseIdList: string[];
}>({
  databaseIdList: [],
});

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const projectStore = useProjectV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const currentUserV1 = useCurrentUserV1();
const currentUserIamPolicy = useCurrentUserIamPolicy();
const pageMode = usePageMode();

const isStandaloneMode = computed(() => pageMode.value === "STANDALONE");

const selectedProjectNames = computed(() => {
  return new Set(props.databases.map((db) => db.project));
});

const assignedDatabases = computed(() => {
  return props.databases.filter((db) => db.project !== DEFAULT_PROJECT_V1_NAME);
});

const getDisabledTooltip = (action: string) => {
  if (selectedProjectNames.value.size > 1) {
    return t("database.batch-action-disabled", {
      action,
    });
  }
  if (selectedProjectNames.value.has(DEFAULT_PROJECT_V1_NAME)) {
    return t("database.batch-action-disabled-for-unassigned", {
      action,
    });
  }
  return "";
};

const selectedProjectUid = computed(() => {
  if (props.projectUid) {
    return props.projectUid;
  }
  if (selectedProjectNames.value.size !== 1) {
    return "";
  }
  const project = [...selectedProjectNames.value][0];
  return projectStore.getProjectByName(project).uid;
});

const canEditDatabase = (
  db: ComposedDatabase,
  requiredProjectPermission: ProjectPermission
): boolean => {
  if (isArchivedDatabaseV1(db)) {
    return false;
  }
  if (currentUserIamPolicy.allowToChangeDatabaseOfProject(db.project)) {
    return true;
  }

  if (
    hasProjectPermissionV2(
      db.projectEntity,
      currentUserV1.value,
      requiredProjectPermission
    )
  ) {
    return true;
  }

  return false;
};

const databaseSupportAlterSchema = computed(() => {
  return props.databases.every((db) => {
    return instanceV1HasAlterSchema(db.instanceEntity);
  });
});

const allowEditSchema = computed(() => {
  return props.databases.every((db) => {
    return canEditDatabase(db, "bb.issues.create");
  });
});

const allowChangeData = computed(() => {
  return props.databases.every((db) => canEditDatabase(db, "bb.issues.create"));
});

const allowTransferProject = computed(() => {
  return props.databases.every((db) =>
    canEditDatabase(db, "bb.projects.update")
  );
});

const allowEditLabels = computed(() => {
  return props.databases.every((db) => {
    const project = db.projectEntity;
    return hasProjectPermissionV2(
      project,
      currentUserV1.value,
      "bb.databases.update"
    );
  });
});

const allowSyncDatabases = computed(() => {
  return props.databases.every((db) => {
    const project = db.projectEntity;
    return hasProjectPermissionV2(
      project,
      currentUserV1.value,
      "bb.databases.sync"
    );
  });
});

const selectedDatabaseUidList = computed(() => {
  return props.databases.map((db) => db.uid);
});

const generateMultiDb = async (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update"
) => {
  if (
    type === "bb.issue.database.schema.update" &&
    allowUsingSchemaEditorV1(props.databases) &&
    !isStandaloneMode.value
  ) {
    schemaEditorContext.value.databaseIdList = [
      ...selectedDatabaseUidList.value,
    ];
    state.showSchemaEditorModal = true;
    return;
  }

  const query: Record<string, any> = {
    template: type,
    name: generateIssueName(
      type,
      props.databases.map((db) => db.databaseName),
      false
    ),
    project: selectedProjectUid.value,
    // The server-side will sort the databases by environment.
    // So we need not to sort them here.
    databaseList: selectedDatabaseUidList.value.join(","),
  };
  const project = useProjectV1Store().getProjectByUID(selectedProjectUid.value);
  router.push({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.name),
      issueSlug: "create",
    },
    query,
  });
};

// TODO: batch request
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
      const requests = props.databases.map((db) => {
        databaseStore.syncDatabase(db.name).then(() => {
          dbSchemaStore.getOrFetchDatabaseMetadata({
            database: db.name,
            view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
            skipCache: true,
          });
        });
      });
      await Promise.all(requests);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("db.successfully-synced-schema"),
      });
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
    await useDatabaseV1Store().transferDatabases(
      assignedDatabases.value,
      DEFAULT_PROJECT_V1_NAME
    );
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

const operationsInProjectDetail = computed(() => !!props.projectUid);

const actions = computed((): DatabaseAction[] => {
  const resp: DatabaseAction[] = [];
  if (!isStandaloneMode.value) {
    resp.push(
      {
        icon: h(RefreshCcwIcon),
        text: t("common.sync"),
        disabled: !allowSyncDatabases.value || props.databases.length < 1,
        click: syncSchema,
        tooltip: (action) => getDisabledTooltip(action),
      },
      {
        icon: h(TagIcon),
        text: t("database.edit-labels"),
        disabled: !allowEditLabels.value || props.databases.length < 1,
        click: () => (state.showLabelEditorDrawer = true),
        tooltip: (action) => {
          if (!allowEditLabels.value) {
            return t("database.batch-action-permission-denied", {
              action,
            });
          }
          return getDisabledTooltip(action);
        },
      }
    );
    if (!operationsInProjectDetail.value) {
      resp.push({
        icon: h(ArrowRightLeftIcon),
        text: t("database.transfer-project"),
        disabled: !allowTransferProject.value || props.databases.length < 1,
        click: () => (state.showTransferOutDatabaseForm = true),
        tooltip: (action) => {
          if (!allowTransferProject.value) {
            return t("database.batch-action-permission-denied", {
              action,
            });
          }
          return getDisabledTooltip(action);
        },
      });
    } else {
      resp.push({
        icon: h(UnlinkIcon),
        text: t("database.unassign"),
        disabled: !allowTransferProject.value || props.databases.length < 1,
        click: () => (state.showUnassignAlert = true),
        tooltip: (action) => {
          if (!allowTransferProject.value) {
            return t("database.batch-action-permission-denied", {
              action,
            });
          }
          return getDisabledTooltip(action);
        },
      });
    }
  }
  resp.unshift(
    {
      icon: h(PenSquareIcon),
      text: t("database.edit-schema"),
      disabled:
        !databaseSupportAlterSchema.value ||
        !allowEditSchema.value ||
        !selectedProjectUid.value ||
        props.databases.length < 1 ||
        selectedProjectNames.value.has(DEFAULT_PROJECT_V1_NAME),
      click: () => generateMultiDb("bb.issue.database.schema.update"),
      tooltip: (action) => {
        if (!databaseSupportAlterSchema.value) {
          return t("database.batch-action-not-support-alter-schema");
        }
        if (!allowEditSchema.value) {
          return t("database.batch-action-permission-denied", {
            action,
          });
        }
        return getDisabledTooltip(action);
      },
    },
    {
      icon: h(PencilIcon),
      text: t("database.change-data"),
      disabled:
        !allowChangeData.value ||
        !selectedProjectUid.value ||
        props.databases.length < 1 ||
        selectedProjectNames.value.has(DEFAULT_PROJECT_V1_NAME),
      click: () => generateMultiDb("bb.issue.database.data.update"),
      tooltip: (action) => {
        if (!allowChangeData.value) {
          return t("database.batch-action-permission-denied", {
            action,
          });
        }
        return getDisabledTooltip(action);
      },
    }
  );

  return resp;
});

const onLabelsApply = async (labelsList: { [key: string]: string }[]) => {
  if (labelsList.length !== props.databases.length) {
    // This should never happen.
    return;
  }

  await Promise.all(
    props.databases.map(async (database, i) => {
      const label = labelsList[i];
      const patch: Database = {
        ...Database.fromPartial(database),
        labels: label,
      };
      await useDatabaseV1Store().updateDatabase({
        database: patch,
        updateMask: ["labels"],
      });
    })
  );

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
