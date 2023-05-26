<template>
  <div
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
    v-bind="$attrs"
  >
    <main class="flex-1 relative overflow-y-auto">
      <!-- Highlight Panel -->
      <div
        class="px-4 pb-4 space-y-2 lg:space-y-0 lg:flex lg:items-center lg:justify-between"
      >
        <div class="flex-1 min-w-0 shrink-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-center">
                <h1
                  class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate flex items-center gap-x-3"
                >
                  {{ database.databaseName }}

                  <ProductionEnvironmentV1Icon
                    :environment="database.instanceEntity.environmentEntity"
                    :tooltip="true"
                    class="w-5 h-5"
                  />

                  <BBBadge
                    v-if="isPITRDatabaseV1(database)"
                    text="PITR"
                    :can-remove="false"
                    class="text-xs"
                  />
                </h1>
              </div>
            </div>
          </div>
          <dl
            class="flex flex-col space-y-1 md:space-y-0 md:flex-row md:flex-wrap"
            data-label="bb-database-detail-info-block"
          >
            <dt class="sr-only">{{ $t("common.environment") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.environment") }}&nbsp;-&nbsp;</span
              >
              <EnvironmentV1Name
                :environment="database.instanceEntity.environmentEntity"
                icon-class="textinfolabel"
              />
            </dd>
            <dt class="sr-only">{{ $t("common.instance") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="ml-1 textlabel"
                >{{ $t("common.instance") }}&nbsp;-&nbsp;</span
              >
              <InstanceV1Name :instance="database.instanceEntity" />
            </dd>
            <dt class="sr-only">{{ $t("common.project") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.project") }}&nbsp;-&nbsp;</span
              >
              <ProjectV1Name
                :project="database.projectEntity"
                hash="#databases"
              />
            </dd>
            <SQLEditorButtonV1
              class="text-sm md:mr-4"
              :database="database"
              :label="true"
              :disabled="!allowQuery"
              @failed="handleGotoSQLEditorFailed"
            />
            <dd
              v-if="hasSchemaDiagramFeature"
              class="flex items-center text-sm md:mr-4 textlabel cursor-pointer hover:text-accent"
              @click.prevent="state.showSchemaDiagram = true"
            >
              <span class="mr-1">{{ $t("schema-diagram.self") }}</span>
              <SchemaDiagramIcon />
            </dd>
            <DatabaseLabelProps
              :labels="database.labels"
              :database="database"
              :allow-edit="allowEditDatabaseLabels"
              @update:labels="updateLabels"
            >
              <template #label="{ label }">
                <span class="textlabel capitalize">
                  {{ hidePrefix(label) }}&nbsp;-&nbsp;
                </span>
              </template>
            </DatabaseLabelProps>
          </dl>
        </div>
        <div
          v-if="allowToChangeDatabase"
          class="flex flex-row justify-end items-center flex-wrap shrink gap-x-2 gap-y-2"
          data-label="bb-database-detail-action-buttons-container"
        >
          <BBSpin v-if="state.syncingSchema" :title="$t('instance.syncing')" />
          <button
            type="button"
            class="btn-normal"
            :disabled="state.syncingSchema"
            @click.prevent="syncDatabaseSchema"
          >
            {{ $t("common.sync-now") }}
          </button>
          <button
            v-if="allowTransferProject"
            type="button"
            class="btn-normal"
            @click.prevent="tryTransferProject"
          >
            <span>{{ $t("database.transfer-project") }}</span>
            <heroicons-outline:switch-horizontal
              class="-mr-1 ml-2 h-5 w-5 text-control-light"
            />
          </button>
          <button
            v-if="allowAlterSchemaOrChangeData"
            type="button"
            class="btn-normal"
            @click="createMigration('bb.issue.database.data.update')"
          >
            <span>{{ $t("database.change-data") }}</span>
          </button>
          <button
            v-if="allowAlterSchema"
            type="button"
            class="btn-normal"
            @click="createMigration('bb.issue.database.schema.update')"
          >
            <span>{{ $t("database.alter-schema") }}</span>
          </button>
        </div>
      </div>
    </main>

    <BBTabFilter
      class="px-3 pb-2 border-b border-block-border"
      :responsive="false"
      :tab-item-list="tabItemList"
      :selected-index="state.selectedIndex"
      data-label="bb-database-detail-tab"
      @select-index="
        (index: number) => {
          selectTab(index);
        }
      "
    />
    <div class="py-6 px-6">
      <template v-if="selectedTabItem?.hash === 'overview'">
        <DatabaseOverviewPanel :database="database" />
      </template>
      <template v-if="selectedTabItem?.hash === 'change-history'">
        <DatabaseMigrationHistoryPanel
          :database="legacyDatabase"
          :allow-edit="allowEdit"
        />
      </template>
      <template v-if="selectedTabItem?.hash === 'backup-and-restore'">
        <DatabaseBackupPanel
          :database="legacyDatabase"
          :allow-admin="allowAdmin"
          :allow-edit="allowEdit"
        />
      </template>
      <template v-if="selectedTabItem?.hash === 'slow-query'">
        <DatabaseSlowQueryPanel :database="database" />
      </template>
      <template v-if="selectedTabItem?.hash === 'settings'">
        <DatabaseSettingsPanel :database="database" />
      </template>
    </div>

    <BBModal
      v-if="state.showTransferDatabaseModal"
      :title="$t('database.transfer-project')"
      @close="state.showTransferDatabaseModal = false"
    >
      <div class="w-112 flex flex-col items-center">
        <div class="col-span-1 w-64">
          <label for="user" class="textlabel">{{ $t("common.project") }}</label>
          <!-- Only allow to transfer database to the project having OWNER role -->
          <ProjectSelect
            id="project"
            class="mt-1"
            name="project"
            :allowed-role-list="['OWNER']"
            :include-default-project="allowTransferToDefaultProject"
            :selected-id="state.currentProjectId"
            @select-project-id="
              (projectId) => {
                state.currentProjectId = projectId;
              }
            "
          />
        </div>
        <SelectDatabaseLabel
          :database="legacyDatabase"
          :target-project-id="state.currentProjectId"
          class="mt-4"
          @next="doTransfer"
        >
          <template #buttons="{ next }">
            <div
              class="w-full pt-4 mt-6 flex justify-end border-t border-block-border"
            >
              <button
                type="button"
                class="btn-normal py-2 px-4"
                @click.prevent="state.showTransferDatabaseModal = false"
              >
                {{ $t("common.cancel") }}
              </button>
              <!--
                We are not allowed to transfer a db either its labels are not valid
                or transferring into its project itself.
              -->
              <button
                type="button"
                class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
                :disabled="state.currentProjectId == legacyDatabase.project.id"
                @click.prevent="next"
              >
                {{ $t("common.transfer") }}
              </button>
            </div>
          </template>
        </SelectDatabaseLabel>
      </div>
    </BBModal>
    <BBModal
      v-if="state.showIncorrectProjectModal"
      :title="$t('common.warning')"
      @close="state.showIncorrectProjectModal = false"
    >
      <div class="col-span-1 w-96">
        {{ $t("database.incorrect-project-warning") }}
      </div>
      <div class="pt-6 flex justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="state.showIncorrectProjectModal = false"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          @click.prevent="
            state.showIncorrectProjectModal = false;
            state.showTransferDatabaseModal = true;
          "
        >
          {{ $t("database.go-to-transfer") }}
        </button>
      </div>
    </BBModal>
  </div>

  <GhostDialog ref="ghostDialog" />

  <BBModal
    v-if="state.showSchemaDiagram"
    :title="$t('schema-diagram.self')"
    class="h-[calc(100vh-40px)] !max-h-[calc(100vh-40px)]"
    header-class="!border-0"
    container-class="flex-1 !pt-0"
    @close="state.showSchemaDiagram = false"
  >
    <div class="w-[80vw] h-full">
      <SchemaDiagram
        :database="legacyDatabase"
        :database-metadata="
          dbSchemaStore.getDatabaseMetadataByDatabaseId(legacyDatabase.id)
        "
      />
    </div>
  </BBModal>

  <SchemaEditorModal
    v-if="state.showSchemaEditorModal"
    :database-id-list="[legacyDatabase.id]"
    alter-type="SINGLE_DB"
    @close="state.showSchemaEditorModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, watch, ref } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import { useI18n } from "vue-i18n";
import { startCase } from "lodash-es";

import ProjectSelect from "@/components/ProjectSelect.vue";
import DatabaseBackupPanel from "@/components/DatabaseBackupPanel.vue";
import DatabaseMigrationHistoryPanel from "@/components/DatabaseMigrationHistoryPanel.vue";
import DatabaseOverviewPanel from "@/components/DatabaseOverviewPanel.vue";
import DatabaseSlowQueryPanel from "@/components/DatabaseSlowQueryPanel.vue";
import {
  DatabaseSettingsPanel,
  SQLEditorButtonV1,
} from "@/components/DatabaseDetail";
import { DatabaseLabelProps } from "@/components/DatabaseLabels";
import { SelectDatabaseLabel } from "@/components/TransferDatabaseForm";
import {
  idFromSlug,
  hasWorkspacePermissionV1,
  hidePrefix,
  allowGhostMigrationV1,
  isPITRDatabaseV1,
  isArchivedDatabaseV1,
  instanceHasBackupRestore,
  instanceHasAlterSchema,
  instanceSupportSlowQuery,
  hasPermissionInProjectV1,
  instanceV1HasAlterSchema,
  isDatabaseV1Accessible,
  allowUsingSchemaEditorV1,
} from "@/utils";
import {
  UNKNOWN_ID,
  DEFAULT_PROJECT_ID,
  Database,
  DatabaseLabel,
  SQLResultSet,
  DEFAULT_PROJECT_V1_NAME,
  ComposedDatabase,
} from "@/types";
import { BBTabFilterItem } from "@/bbkit/types";
import { GhostDialog } from "@/components/AlterSchemaPrepForm";
import { SchemaDiagram, SchemaDiagramIcon } from "@/components/SchemaDiagram";
import {
  pushNotification,
  useCurrentUserIamPolicy,
  useCurrentUserV1,
  useDatabaseStore,
  useDatabaseV1Store,
  useDBSchemaStore,
  useGracefulRequest,
  useSQLStore,
} from "@/store";
import { usePolicyByParentAndType } from "@/store/modules/v1/policy";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import {
  EnvironmentV1Name,
  InstanceV1Name,
  ProductionEnvironmentV1Icon,
  ProjectV1Name,
} from "@/components/v2";
import { TenantMode } from "@/types/proto/v1/project_service";
import { State } from "@/types/proto/v1/common";

type DatabaseTabItem = {
  name: string;
  hash: string;
};

interface LocalState {
  showTransferDatabaseModal: boolean;
  showIncorrectProjectModal: boolean;
  showSchemaEditorModal: boolean;
  currentProjectId: string;
  selectedIndex: number;
  syncingSchema: boolean;
  showSchemaDiagram: boolean;
}

const props = defineProps({
  databaseSlug: {
    required: true,
    type: String,
  },
});

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseStore();
const databaseV1Store = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaStore();
const sqlStore = useSQLStore();
const ghostDialog = ref<InstanceType<typeof GhostDialog>>();

const databaseTabItemList = computed((): DatabaseTabItem[] => {
  if (!allowToChangeDatabase.value) {
    return [{ name: t("common.overview"), hash: "overview" }];
  }

  return [
    { name: t("common.overview"), hash: "overview" },
    { name: t("change-history.self"), hash: "change-history" },
    { name: t("common.backup-and-restore"), hash: "backup-and-restore" },
    { name: startCase(t("slow-query.slow-queries")), hash: "slow-query" },
    { name: t("common.settings"), hash: "settings" },
  ];
});

const state = reactive<LocalState>({
  showTransferDatabaseModal: false,
  showIncorrectProjectModal: false,
  showSchemaEditorModal: false,
  currentProjectId: String(UNKNOWN_ID),
  selectedIndex: 0,
  syncingSchema: false,
  showSchemaDiagram: false,
});

const currentUserV1 = useCurrentUserV1();
const currentUserIamPolicy = useCurrentUserIamPolicy();

const legacyDatabase = computed((): Database => {
  return databaseStore.getDatabaseById(idFromSlug(props.databaseSlug));
});
const database = computed(() => {
  return databaseV1Store.getDatabaseByUID(
    String(idFromSlug(props.databaseSlug))
  );
});
const project = computed(() => database.value.projectEntity);

const allowToChangeDatabase = computed(() => {
  return currentUserIamPolicy.allowToChangeDatabaseOfProject(
    project.value.name
  );
});

const hasSchemaDiagramFeature = computed((): boolean => {
  return instanceV1HasAlterSchema(database.value.instanceEntity);
});

const accessControlPolicy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: database.value.name,
    policyType: PolicyType.ACCESS_CONTROL,
  }))
);
const allowQuery = computed(() => {
  const policy = accessControlPolicy.value;
  const list = policy ? [policy] : [];
  return isDatabaseV1Accessible(database.value, list, currentUserV1.value);
});

// Project can be transferred if meets either of the condition below:
// - Database is in default project
// - Workspace role can manage instance
// - Project role can transfer database
const allowTransferProject = computed(() => {
  if (isArchivedDatabaseV1(database.value)) {
    return false;
  }

  if (database.value.project === DEFAULT_PROJECT_V1_NAME) {
    return true;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-project",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProjectV1(
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.transfer-database"
    )
  ) {
    return true;
  }

  return false;
});

const allowTransferToDefaultProject = computed(() => {
  if (database.value.project === DEFAULT_PROJECT_V1_NAME) {
    return true;
  }

  // Allow to transfer a database to DEFAULT project only if the current user
  // can manage all projects.
  // AKA DBA or workspace owner.
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-project",
    currentUserV1.value.userRole
  );
});

// Database can be admined if meets either of the condition below:
// - Workspace role can manage instance
// - Project role can admin database
//
// The admin operation includes
// - Edit database label
// - Enable/disable backup
const allowAdmin = computed(() => {
  if (isArchivedDatabaseV1(database.value)) {
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
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.admin-database"
    )
  ) {
    return true;
  }
  return false;
});

// Database can be edited if meets either of the condition below:
// - Workspace role can manage instance
// - Project role can change database
//
// The edit operation includes
// - Take manual backup
const allowEdit = computed(() => {
  if (isArchivedDatabaseV1(database.value)) {
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
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.change-database"
    )
  ) {
    return true;
  }
  return false;
});

const allowAlterSchemaOrChangeData = computed(() => {
  if (legacyDatabase.value.project.id === DEFAULT_PROJECT_ID) {
    return false;
  }
  return allowEdit.value;
});

const allowAlterSchema = computed(() => {
  return (
    allowAlterSchemaOrChangeData.value &&
    instanceHasAlterSchema(legacyDatabase.value.instance)
  );
});

const allowEditDatabaseLabels = computed((): boolean => {
  // only allowed to edit database labels when allowAdmin
  return allowAdmin.value;
});

const availableDatabaseTabItemList = computed(() => {
  const db = legacyDatabase.value;
  return databaseTabItemList.value.filter((item) => {
    if (item.hash === "backup-and-restore") {
      return instanceHasBackupRestore(db.instance);
    }
    if (item.hash === "slow-query") {
      return instanceSupportSlowQuery(db.instance);
    }
    return true;
  });
});

const tabItemList = computed((): BBTabFilterItem[] => {
  return availableDatabaseTabItemList.value.map((item) => {
    return { title: item.name, alert: false };
  });
});

const tryTransferProject = () => {
  state.currentProjectId = project.value.uid;
  state.showTransferDatabaseModal = true;
};

// 'normal' -> normal migration
// 'online' -> online migration
// false -> user clicked cancel button
const isUsingGhostMigration = async (databaseList: ComposedDatabase[]) => {
  if (project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED) {
    // Not available for tenant mode now.
    return "normal";
  }

  // check if all selected databases supports gh-ost
  if (allowGhostMigrationV1(databaseList)) {
    // open the dialog to ask the user
    const { result, mode } = await ghostDialog.value!.open();
    if (!result) {
      return false; // return false when user clicked the cancel button
    }
    return mode;
  }

  // fallback to normal
  return "normal";
};

const createMigration = async (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update"
) => {
  type AlterMode = "online" | "normal" | false;
  let mode: AlterMode = "normal";
  if (type === "bb.issue.database.schema.update") {
    if (
      database.value.syncState === State.ACTIVE &&
      allowUsingSchemaEditorV1([database.value])
    ) {
      state.showSchemaEditorModal = true;
      return;
    }

    // Check and show a normal/online selection modal dialog if needed.
    mode = await isUsingGhostMigration([database.value]);
  }
  if (mode === false) return;

  // Create a user friendly default issue name
  const issueNameParts: string[] = [];
  issueNameParts.push(`[${database.value.databaseName}]`);
  if (mode === "online") {
    issueNameParts.push("Online schema change");
  } else {
    issueNameParts.push(
      type === "bb.issue.database.schema.update"
        ? `Alter schema`
        : `Change data`
    );
  }
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  const query: Record<string, any> = {
    template: type,
    name: issueNameParts.join(" "),
    project: project.value.uid,
    databaseList: database.value.uid,
  };
  if (mode === "online") {
    query.ghost = "1";
  }

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  });
};

const updateProject = (newProjectId: string, labels?: DatabaseLabel[]) => {
  databaseStore
    .transferProject({
      database: legacyDatabase.value,
      projectId: newProjectId,
      labels,
    })
    .then((updatedDatabase) => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "database.successfully-transferred-updateddatabase-name-to-project-updateddatabase-project-name",
          [updatedDatabase.name, updatedDatabase.project.name]
        ),
      });
    });
};

const updateLabels = (labels: Record<string, string>) => {
  useGracefulRequest(async () => {
    const databasePatch = { ...database.value };
    databasePatch.labels = labels;
    await databaseV1Store.updateDatabase({
      database: databasePatch,
      updateMask: ["labels"],
    });
  });
};

const selectedTabItem = computed(() => {
  return availableDatabaseTabItemList.value[state.selectedIndex];
});

const selectTab = (index: number) => {
  const item = availableDatabaseTabItemList.value[index];
  state.selectedIndex = index;
  router.replace({
    name: "workspace.database.detail",
    hash: "#" + item.hash,
  });
};

const selectDatabaseTabOnHash = () => {
  if (router.currentRoute.value.hash) {
    for (let i = 0; i < availableDatabaseTabItemList.value.length; i++) {
      if (
        availableDatabaseTabItemList.value[i].hash ==
        router.currentRoute.value.hash.slice(1)
      ) {
        selectTab(i);
        break;
      }
    }
  } else {
    selectTab(0);
  }
};

const handleGotoSQLEditorFailed = () => {
  state.currentProjectId = String(legacyDatabase.value.project.id);
  state.showIncorrectProjectModal = true;
};

onMounted(() => {
  selectDatabaseTabOnHash();
});

watch(
  () => router.currentRoute.value.hash,
  () => {
    if (router.currentRoute.value.name == "workspace.database.detail") {
      selectDatabaseTabOnHash();
    }
  }
);

const doTransfer = (labels: DatabaseLabel[]) => {
  updateProject(state.currentProjectId, labels);
  state.showTransferDatabaseModal = false;
};

const syncDatabaseSchema = () => {
  state.syncingSchema = true;
  sqlStore
    .syncDatabaseSchema(legacyDatabase.value.id)
    .then((resultSet: SQLResultSet) => {
      state.syncingSchema = false;
      if (resultSet.error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t(
            "db.failed-to-sync-schema-for-database-database-value-name",
            [legacyDatabase.value.name]
          ),
          description: resultSet.error,
        });
      } else {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t(
            "db.successfully-synced-schema-for-database-database-value-name",
            [legacyDatabase.value.name]
          ),
          description: resultSet.error,
        });
      }
      useDBSchemaStore().getOrFetchDatabaseMetadataById(
        legacyDatabase.value.id,
        true // skip cache
      );
    })
    .catch(() => {
      state.syncingSchema = false;
    });
};
</script>
