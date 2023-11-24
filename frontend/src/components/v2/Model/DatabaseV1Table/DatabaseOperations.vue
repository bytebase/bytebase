<template>
  <div
    v-bind="$attrs"
    class="flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-2 text-main gap-y-2"
  >
    <div class="flex items-center">
      <button
        v-if="!isStandaloneMode"
        class="w-7 h-7 p-1 mr-3 rounded cursor-pointer"
        @click.prevent="$emit('dismiss')"
      >
        <heroicons-outline:x class="w-5 h-5" />
      </button>
      {{
        $t("database.selected-n-databases", {
          n: databases.length,
        })
      }}
    </div>
    <div class="flex items-center gap-x-4 text-sm ml-5 text-accent">
      <template v-for="action in actions" :key="action.text">
        <NTooltip :disabled="!action.disabled">
          <template #trigger>
            <button
              :disabled="action.disabled"
              class="flex items-center gap-x-1 hover:text-accent-hover disabled:text-control-light disabled:cursor-not-allowed"
              @click="action.click"
            >
              <component :is="action.icon" class="h-4 w-4" />
              {{ action.text }}
            </button>
          </template>
          <span class="w-56 text-sm">
            {{
              $t("database.batch-action-disabled", {
                action: action.text.toLowerCase(),
              })
            }}
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

  <Drawer
    v-if="selectedProjectUid"
    :show="state.showTransferOutDatabaseForm"
    :auto-focus="true"
    @close="state.showTransferOutDatabaseForm = false"
  >
    <TransferOutDatabaseForm
      :project-id="selectedProjectUid"
      :selected-database-uid-list="selectedDatabaseUidList"
      @dismiss="state.showTransferOutDatabaseForm = false"
    />
  </Drawer>

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
</template>

<script lang="ts" setup>
import { computed, h, VNode, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import PencilIcon from "~icons/heroicons-outline/pencil";
import PencilAltIcon from "~icons/heroicons-outline/pencil-alt";
import RefreshIcon from "~icons/heroicons-outline/refresh";
import SwitchHorizontalIcon from "~icons/heroicons-outline/switch-horizontal";
import TagIcon from "~icons/heroicons-outline/tag";
import { Drawer } from "@/components/v2";
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
import { ComposedDatabase, DEFAULT_PROJECT_V1_NAME } from "@/types";
import { Database } from "@/types/proto/v1/database_service";
import {
  hasWorkspacePermissionV1,
  hasPermissionInProjectV1,
  isArchivedDatabaseV1,
  instanceV1HasAlterSchema,
  allowUsingSchemaEditorV1,
  generateIssueName,
} from "@/utils";
import { ProjectPermissionType } from "@/utils/role";

interface DatabaseAction {
  icon: VNode;
  text: string;
  disabled: boolean;
  click: () => void;
}

interface LocalState {
  loading: boolean;
  showSchemaEditorModal: boolean;
  showTransferOutDatabaseForm: boolean;
  showLabelEditorDrawer: boolean;
}

const props = defineProps<{
  databases: ComposedDatabase[];
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();

const state = reactive<LocalState>({
  loading: false,
  showSchemaEditorModal: false,
  showTransferOutDatabaseForm: false,
  showLabelEditorDrawer: false,
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

const selectedProjectUid = computed(() => {
  if (selectedProjectNames.value.size !== 1) {
    return "";
  }
  const project = [...selectedProjectNames.value][0];
  return projectStore.getProjectByName(project).uid;
});

const canEditDatabase = (
  db: ComposedDatabase,
  requiredProjectPermission: ProjectPermissionType
): boolean => {
  if (isArchivedDatabaseV1(db)) {
    return false;
  }
  if (!currentUserIamPolicy.allowToChangeDatabaseOfProject(db.project)) {
    return false;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-instance",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProjectV1(
      db.projectEntity.iamPolicy,
      currentUserV1.value,
      requiredProjectPermission
    )
  ) {
    return true;
  }

  return false;
};

const allowEditSchema = computed(() => {
  return props.databases.every((db) => {
    return (
      canEditDatabase(db, "bb.permission.project.change-database") &&
      instanceV1HasAlterSchema(db.instanceEntity)
    );
  });
});

const allowChangeData = computed(() => {
  return props.databases.every((db) =>
    canEditDatabase(db, "bb.permission.project.change-database")
  );
});

const allowTransferProject = computed(() => {
  return props.databases.every((db) =>
    canEditDatabase(db, "bb.permission.project.transfer-database")
  );
});

const allowEditLabels = computed(() => {
  return props.databases.every((db) => {
    const project = db.projectEntity;
    return (
      hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-label",
        currentUserV1.value.userRole
      ) ||
      hasPermissionInProjectV1(
        project.iamPolicy,
        currentUserV1.value,
        "bb.permission.project.manage-general"
      )
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
  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
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

const actions = computed((): DatabaseAction[] => {
  const resp: DatabaseAction[] = [];
  if (!isStandaloneMode.value) {
    resp.push({
      icon: h(RefreshIcon),
      text: t("common.sync"),
      disabled: props.databases.length < 1,
      click: syncSchema,
    });
    if (allowTransferProject.value) {
      resp.unshift({
        icon: h(SwitchHorizontalIcon),
        text: t("database.transfer-project"),
        disabled: !selectedProjectUid.value,
        click: () => (state.showTransferOutDatabaseForm = true),
      });
    }
    if (allowEditLabels.value) {
      resp.push({
        icon: h(TagIcon),
        text: t("database.edit-labels"),
        disabled: false,
        click: () => (state.showLabelEditorDrawer = true),
      });
    }
  }
  if (!selectedProjectNames.value.has(DEFAULT_PROJECT_V1_NAME)) {
    if (allowChangeData.value) {
      resp.unshift({
        icon: h(PencilIcon),
        text: t("database.change-data"),
        disabled: !selectedProjectUid.value,
        click: () => generateMultiDb("bb.issue.database.data.update"),
      });
    }
    if (allowEditSchema.value) {
      resp.unshift({
        icon: h(PencilAltIcon),
        text: t("database.edit-schema"),
        disabled: !selectedProjectUid.value,
        click: () => generateMultiDb("bb.issue.database.schema.update"),
      });
    }
  }
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
