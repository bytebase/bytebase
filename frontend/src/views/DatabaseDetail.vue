<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
    <main class="flex-1 relative overflow-y-auto">
      <!-- Highlight Panel -->
      <div
        class="px-4 pb-4 space-y-2 md:space-y-0 md:flex md:items-center md:justify-between"
      >
        <div class="flex-1 min-w-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-center">
                <h1
                  class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate flex items-center gap-x-3"
                >
                  {{ database.name }}

                  <BBBadge
                    v-if="isPITRDatabase(database)"
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
              <router-link
                :to="`/environment/${environmentSlug(
                  database.instance.environment
                )}`"
                class="normal-link"
              >
                {{ environmentName(database.instance.environment) }}
              </router-link>
            </dd>
            <dt class="sr-only">{{ $t("common.instance") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <InstanceEngineIcon :instance="database.instance" />
              <span class="ml-1 textlabel"
                >{{ $t("common.instance") }}&nbsp;-&nbsp;</span
              >
              <router-link
                :to="`/instance/${instanceSlug(database.instance)}`"
                class="normal-link"
                >{{ instanceName(database.instance) }}</router-link
              >
            </dd>
            <dt class="sr-only">{{ $t("common.project") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.project") }}&nbsp;-&nbsp;</span
              >
              <router-link
                :to="`/project/${projectSlug(database.project)}`"
                class="normal-link"
                >{{ projectName(database.project) }}</router-link
              >
            </dd>
            <template v-if="database.sourceBackup">
              <dt class="sr-only">{{ $t("db.parent") }}</dt>
              <dd class="flex items-center text-sm md:mr-4 tooltip-wrapper">
                <span class="textlabel">{{
                  $t("database.restored-from")
                }}</span>
                <router-link
                  :to="`/db/${database.sourceBackup.databaseId}`"
                  class="normal-link"
                >
                  <!-- Do not display the name of the backup's database because that requires a fetch  -->
                  <span class="tooltip">
                    {{
                      $t(
                        "database.database-name-is-restored-from-another-database-backup",
                        [database.name]
                      )
                    }}
                  </span>
                  {{ $t("database.database-backup") }}
                </router-link>
              </dd>
            </template>
            <dd
              class="flex items-center text-sm md:mr-4 cursor-pointer textlabel hover:text-accent"
              @click.prevent="gotoSQLEditor"
            >
              <span class="mr-1">{{ $t("sql-editor.self") }}</span>
              <heroicons-outline:terminal class="w-4 h-4" />
            </dd>
            <DatabaseLabelProps
              :label-list="database.labels"
              :database="database"
              :allow-edit="allowEditDatabaseLabels"
              @update:label-list="updateLabels"
            >
              <template #label="{ label }">
                <span class="textlabel capitalize">
                  {{ hidePrefix(label.key) }}&nbsp;-&nbsp;
                </span>
              </template>
            </DatabaseLabelProps>
          </dl>
        </div>
        <div
          class="flex items-center gap-x-2"
          data-label="bb-database-detail-action-buttons-container"
        >
          <BBSpin v-if="state.syncingSchema" :title="$t('instance.syncing')" />
          <button
            type="button"
            class="btn-normal"
            @click.prevent="syncDatabaseSchema"
          >
            {{ $t("common.sync-now") }}
          </button>
          <button
            v-if="allowChangeProject"
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
            v-if="allowEdit"
            type="button"
            class="btn-normal"
            @click="createMigration('bb.issue.database.data.update')"
          >
            <span>{{ changeDataText }}</span>
            <heroicons-outline:external-link
              v-if="database.project.workflowType == 'VCS'"
              class="-mr-1 ml-2 h-5 w-5 text-control-light"
            />
          </button>
          <button
            v-if="allowEdit"
            type="button"
            class="btn-normal"
            @click="createMigration('bb.issue.database.schema.update')"
          >
            <span>{{ alterSchemaText }}</span>
            <heroicons-outline:external-link
              v-if="database.project.workflowType == 'VCS'"
              class="-mr-1 ml-2 h-5 w-5 text-control-light"
            />
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
      <template v-if="state.selectedIndex == OVERVIEW_TAB">
        <DatabaseOverviewPanel :database="database" />
      </template>
      <template v-if="state.selectedIndex == MIGRATION_HISTORY_TAB">
        <DatabaseMigrationHistoryPanel
          :database="database"
          :allow-edit="allowEdit"
        />
      </template>
      <template v-if="state.selectedIndex == BACKUP_TAB">
        <DatabaseBackupPanel
          :database="database"
          :allow-admin="allowAdmin"
          :allow-edit="allowEdit"
        />
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
            :include-default-project="true"
            :selected-id="state.editingProjectId"
            @select-project-id="
              (projectId) => {
                state.editingProjectId = projectId;
              }
            "
          />
        </div>
        <SelectDatabaseLabel
          :database="database"
          :target-project-id="state.editingProjectId"
          class="mt-4"
          @next="doTransfer"
        >
          <template #buttons="{ next, valid }">
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
                :disabled="
                  !valid || state.editingProjectId == database.project.id
                "
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
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, watch, ref } from "vue";
import { useRouter } from "vue-router";
import ProjectSelect from "@/components/ProjectSelect.vue";
import DatabaseBackupPanel from "@/components/DatabaseBackupPanel.vue";
import DatabaseMigrationHistoryPanel from "@/components/DatabaseMigrationHistoryPanel.vue";
import DatabaseOverviewPanel from "@/components/DatabaseOverviewPanel.vue";
import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";
import { DatabaseLabelProps } from "@/components/DatabaseLabels";
import { SelectDatabaseLabel } from "@/components/TransferDatabaseForm";
import {
  idFromSlug,
  isDBAOrOwner,
  connectionSlug,
  hidePrefix,
  allowGhostMigration,
  isPITRDatabase,
} from "@/utils";
import {
  ProjectId,
  UNKNOWN_ID,
  DEFAULT_PROJECT_ID,
  Repository,
  baseDirectoryWebUrl,
  Database,
  DatabaseLabel,
  SQLResultSet,
} from "@/types";
import { BBTabFilterItem } from "@/bbkit/types";
import { useI18n } from "vue-i18n";
import { GhostDialog } from "@/components/AlterSchemaPrepForm";
import {
  pushNotification,
  useCurrentUser,
  useDatabaseStore,
  useRepositoryStore,
  useSQLStore,
} from "@/store";
import dayjs from "dayjs";

const OVERVIEW_TAB = 0;
const MIGRATION_HISTORY_TAB = 1;
const BACKUP_TAB = 2;

type DatabaseTabItem = {
  name: string;
  hash: string;
};

interface LocalState {
  showTransferDatabaseModal: boolean;
  showIncorrectProjectModal: boolean;
  editingProjectId: ProjectId;
  selectedIndex: number;
  syncingSchema: boolean;
}

const props = defineProps({
  databaseSlug: {
    required: true,
    type: String,
  },
});

const databaseStore = useDatabaseStore();
const repositoryStore = useRepositoryStore();
const sqlStore = useSQLStore();
const router = useRouter();
const { t } = useI18n();
const ghostDialog = ref<InstanceType<typeof GhostDialog>>();

const databaseTabItemList: DatabaseTabItem[] = [
  { name: t("common.overview"), hash: "overview" },
  { name: t("migration-history.self"), hash: "migration-history" },
  { name: t("common.backup-and-restore"), hash: "backup-and-restore" },
];

const state = reactive<LocalState>({
  showTransferDatabaseModal: false,
  showIncorrectProjectModal: false,
  editingProjectId: UNKNOWN_ID,
  selectedIndex: OVERVIEW_TAB,
  syncingSchema: false,
});

const currentUser = useCurrentUser();

const database = computed((): Database => {
  return databaseStore.getDatabaseById(idFromSlug(props.databaseSlug));
});

const isCurrentUserDBAOrOwner = computed((): boolean => {
  return isDBAOrOwner(currentUser.value.role);
});

// Project can be transferred if meets either of the condition below:
// - Database is in default project
// - Workspace owner, dba
// - db's project owner
const allowChangeProject = computed(() => {
  if (database.value.project.id == DEFAULT_PROJECT_ID) {
    return true;
  }

  if (isCurrentUserDBAOrOwner.value) {
    return true;
  }

  for (const member of database.value.project.memberList) {
    if (member.role == "OWNER" && member.principal.id == currentUser.value.id) {
      return true;
    }
  }

  return false;
});

// Database can be admined if meets either of the condition below:
// - Workspace owner, dba
// - db's project owner
//
// The admin operation includes
// - Transfer project
// - Enable/disable backup
const allowAdmin = computed(() => {
  if (isCurrentUserDBAOrOwner.value) {
    return true;
  }

  for (const member of database.value.project.memberList) {
    if (member.role == "OWNER" && member.principal.id == currentUser.value.id) {
      return true;
    }
  }
  return false;
});

// Database can be edited if meets either of the condition below:
// - Workspace owner, dba
// - db's project member
//
// The edit operation includes
// - Take manual backup
const allowEdit = computed(() => {
  if (isCurrentUserDBAOrOwner.value) {
    return true;
  }

  for (const member of database.value.project.memberList) {
    if (member.principal.id == currentUser.value.id) {
      return true;
    }
  }
  return false;
});

const allowEditDatabaseLabels = computed((): boolean => {
  // only allowed to edit database labels when allowAdmin
  return allowAdmin.value;
});

const alterSchemaText = computed(() => {
  if (database.value.project.workflowType == "VCS") {
    return t("database.alter-schema-in-vcs");
  }
  return t("database.alter-schema");
});

const changeDataText = computed(() => {
  if (database.value.project.workflowType == "VCS") {
    return t("database.change-data-in-vcs");
  }
  return t("database.change-data");
});

const tabItemList = computed((): BBTabFilterItem[] => {
  return databaseTabItemList.map((item) => {
    return { title: item.name, alert: false };
  });
});

const tryTransferProject = () => {
  state.editingProjectId = database.value.project.id;
  state.showTransferDatabaseModal = true;
};

// 'normal' -> normal migration
// 'online' -> online migration
// false -> user clicked cancel button
const isUsingGhostMigration = async (databaseList: Database[]) => {
  if (database.value.project.tenantMode === "TENANT") {
    // Not available for tenant mode now.
    return "normal";
  }

  // check if all selected databases supports gh-ost
  if (allowGhostMigration(databaseList)) {
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
  if (database.value.project.workflowType == "UI") {
    let mode: "online" | "normal" | false = "normal";
    if (type === "bb.issue.database.schema.update") {
      // Check and show a normal/online selection modal dialog if needed.
      mode = await isUsingGhostMigration([database.value]);
    }
    if (mode === false) return;

    // Create a user friendly default issue name
    const issueNameParts: string[] = [];
    issueNameParts.push(`[${database.value.name}]`);
    if (mode === "online") {
      issueNameParts.push("Online schema change");
    } else {
      issueNameParts.push(
        type === "bb.issue.database.schema.update"
          ? `Alter schema`
          : `Change data`
      );
    }
    issueNameParts.push(dayjs().format("@MM-DD HH:mm"));

    const query: Record<string, any> = {
      template: type,
      name: issueNameParts.join(" "),
      project: database.value.project.id,
      databaseList: database.value.id,
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
  } else if (database.value.project.workflowType == "VCS") {
    repositoryStore
      .fetchRepositoryByProjectId(database.value.project.id)
      .then((repository: Repository) => {
        window.open(baseDirectoryWebUrl(repository), "_blank");
      });
  }
};

const updateProject = (newProjectId: ProjectId, labels?: DatabaseLabel[]) => {
  databaseStore
    .transferProject({
      databaseId: database.value.id,
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

const updateLabels = (labels: DatabaseLabel[]) => {
  databaseStore.patchDatabaseLabels({
    databaseId: database.value.id,
    labels,
  });
};

const selectTab = (index: number) => {
  state.selectedIndex = index;
  router.replace({
    name: "workspace.database.detail",
    hash: "#" + databaseTabItemList[index].hash,
  });
};

const selectDatabaseTabOnHash = () => {
  if (router.currentRoute.value.hash) {
    for (let i = 0; i < databaseTabItemList.length; i++) {
      if (
        databaseTabItemList[i].hash == router.currentRoute.value.hash.slice(1)
      ) {
        selectTab(i);
        break;
      }
    }
  } else {
    selectTab(OVERVIEW_TAB);
  }
};

const gotoSQLEditor = () => {
  // SQL editors can only query databases in the projects available to the user.
  if (
    database.value.projectId === UNKNOWN_ID ||
    database.value.projectId === DEFAULT_PROJECT_ID
  ) {
    state.editingProjectId = database.value.project.id;
    state.showIncorrectProjectModal = true;
  } else {
    router.push({
      name: "sql-editor.detail",
      params: {
        connectionSlug: connectionSlug(database.value),
      },
    });
  }
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
  updateProject(state.editingProjectId, labels);
  state.showTransferDatabaseModal = false;
};

const syncDatabaseSchema = () => {
  state.syncingSchema = true;
  sqlStore
    .syncDatabaseSchema(database.value.id)
    .then((resultSet: SQLResultSet) => {
      state.syncingSchema = false;
      if (resultSet.error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t(
            "db.failed-to-sync-schema-for-database-database-value-name",
            [database.value.name]
          ),
          description: resultSet.error,
        });
      } else {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t(
            "db.successfully-synced-schema-for-database-database-value-name",
            [database.value.name]
          ),
          description: resultSet.error,
        });
      }
    })
    .catch(() => {
      state.syncingSchema = false;
    });
};
</script>
