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
                  class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate"
                >
                  {{ database.name }}
                </h1>
              </div>
            </div>
          </div>
          <dl
            class="flex flex-col space-y-1 md:space-y-0 md:flex-row md:flex-wrap"
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
              @click.prevent="gotoSqlEditor"
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
        <div class="flex items-center space-x-2">
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
          <BBTooltipButton
            v-if="allowEdit"
            type="normal"
            tooltip-mode="DISABLED-ONLY"
            :disabled="!allowMigrate"
            @click="changeData"
          >
            <span>{{ changeDataText }}</span>
            <heroicons-outline:external-link
              v-if="database.project.workflowType == 'VCS'"
              class="-mr-1 ml-2 h-5 w-5 text-control-light"
            />
            <template v-if="!allowMigrate" #tooltip>
              <div class="w-48 whitespace-pre-wrap">
                {{
                  $t("issue.not-allowed-to-single-database-in-tenant-mode", {
                    operation: changeDataText.toLowerCase(),
                  })
                }}
              </div>
            </template>
          </BBTooltipButton>
          <BBTooltipButton
            v-if="allowEdit"
            type="normal"
            tooltip-mode="DISABLED-ONLY"
            :disabled="!allowMigrate"
            @click="alterSchema"
          >
            <span>{{ alterSchemaText }}</span>
            <heroicons-outline:external-link
              v-if="database.project.workflowType == 'VCS'"
              class="-mr-1 ml-2 h-5 w-5 text-control-light"
            />
            <template v-if="!allowMigrate" #tooltip>
              <div class="w-48 whitespace-pre-wrap">
                {{
                  $t("issue.not-allowed-to-single-database-in-tenant-mode", {
                    operation: alterSchemaText.toLowerCase(),
                  })
                }}
              </div>
            </template>
          </BBTooltipButton>
        </div>
      </div>
    </main>

    <BBTabFilter
      class="px-3 pb-2 border-b border-block-border"
      :responsive="false"
      :tab-item-list="tabItemList"
      :selected-index="state.selectedIndex"
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
</template>

<script lang="ts">
import { computed, onMounted, reactive, watch, defineComponent } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import ProjectSelect from "../components/ProjectSelect.vue";
import DatabaseBackupPanel from "../components/DatabaseBackupPanel.vue";
import DatabaseMigrationHistoryPanel from "../components/DatabaseMigrationHistoryPanel.vue";
import DatabaseOverviewPanel from "../components/DatabaseOverviewPanel.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";
import { DatabaseLabelProps } from "../components/DatabaseLabels";
import { SelectDatabaseLabel } from "../components/TransferDatabaseForm";
import { idFromSlug, isDBAOrOwner, connectionSlug, hidePrefix } from "../utils";
import {
  ProjectId,
  UNKNOWN_ID,
  DEFAULT_PROJECT_ID,
  Repository,
  baseDirectoryWebUrl,
  Database,
  DatabaseLabel,
} from "../types";
import { BBTabFilterItem } from "../bbkit/types";
import { useI18n } from "vue-i18n";
import { useCurrentUser } from "@/store";

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
}

export default defineComponent({
  name: "DatabaseDetail",
  components: {
    ProjectSelect,
    DatabaseOverviewPanel,
    DatabaseMigrationHistoryPanel,
    DatabaseBackupPanel,
    InstanceEngineIcon,
    DatabaseLabelProps,
    SelectDatabaseLabel,
  },
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
  },
  setup(props) {
    const store = useStore();
    const router = useRouter();
    const { t } = useI18n();

    const databaseTabItemList: DatabaseTabItem[] = [
      { name: t("common.overview"), hash: "overview" },
      { name: t("migration-history.self"), hash: "migration-history" },
      { name: t("common.backups"), hash: "backup" },
    ];

    const state = reactive<LocalState>({
      showTransferDatabaseModal: false,
      showIncorrectProjectModal: false,
      editingProjectId: UNKNOWN_ID,
      selectedIndex: OVERVIEW_TAB,
    });

    const currentUser = useCurrentUser();

    const database = computed((): Database => {
      return store.getters["database/databaseById"](
        idFromSlug(props.databaseSlug)
      );
    });

    const isTenantProject = computed(() => {
      return database.value.project.tenantMode === "TENANT";
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
        if (
          member.role == "OWNER" &&
          member.principal.id == currentUser.value.id
        ) {
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
        if (
          member.role == "OWNER" &&
          member.principal.id == currentUser.value.id
        ) {
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

    const allowMigrate = computed(() => {
      if (!allowEdit.value) return false;

      // Migrating single database in tenant mode is not allowed
      // Since this will probably cause different migration version across a group of tenant databases
      return database.value.project.tenantMode === "DISABLED";
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

    const alterSchema = () => {
      if (database.value.project.workflowType == "UI") {
        router.push({
          name: "workspace.issue.detail",
          params: {
            issueSlug: "new",
          },
          query: {
            template: "bb.issue.database.schema.update",
            name: `[${database.value.name}] Alter schema`,
            project: database.value.project.id,
            databaseList: database.value.id,
          },
        });
      } else if (database.value.project.workflowType == "VCS") {
        store
          .dispatch(
            "repository/fetchRepositoryByProjectId",
            database.value.project.id
          )
          .then((repository: Repository) => {
            window.open(baseDirectoryWebUrl(repository), "_blank");
          });
      }
    };

    const changeData = () => {
      if (database.value.project.workflowType == "UI") {
        router.push({
          name: "workspace.issue.detail",
          params: {
            issueSlug: "new",
          },
          query: {
            template: "bb.issue.database.data.update",
            name: `[${database.value.name}] Change data`,
            project: database.value.project.id,
            databaseList: database.value.id,
          },
        });
      } else if (database.value.project.workflowType == "VCS") {
        store
          .dispatch(
            "repository/fetchRepositoryByProjectId",
            database.value.project.id
          )
          .then((repository: Repository) => {
            window.open(baseDirectoryWebUrl(repository), "_blank");
          });
      }
    };

    const updateProject = (
      newProjectId: ProjectId,
      labels?: DatabaseLabel[]
    ) => {
      store
        .dispatch("database/transferProject", {
          databaseId: database.value.id,
          projectId: newProjectId,
          labels,
        })
        .then((updatedDatabase) => {
          store.dispatch("notification/pushNotification", {
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
      store.dispatch("database/patchDatabaseLabels", {
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
            databaseTabItemList[i].hash ==
            router.currentRoute.value.hash.slice(1)
          ) {
            selectTab(i);
            break;
          }
        }
      } else {
        selectTab(OVERVIEW_TAB);
      }
    };

    const gotoSqlEditor = () => {
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

    return {
      OVERVIEW_TAB,
      MIGRATION_HISTORY_TAB,
      BACKUP_TAB,
      state,
      isTenantProject,
      database,
      allowChangeProject,
      allowAdmin,
      allowEdit,
      allowMigrate,
      allowEditDatabaseLabels,
      tabItemList,
      tryTransferProject,
      alterSchema,
      changeData,
      alterSchemaText,
      changeDataText,
      updateProject,
      updateLabels,
      selectTab,
      gotoSqlEditor,
      doTransfer,
      hidePrefix,
    };
  },
});
</script>
