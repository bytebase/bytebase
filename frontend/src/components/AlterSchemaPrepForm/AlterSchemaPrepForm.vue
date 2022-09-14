<template>
  <div class="mx-4 space-y-4 max-w-min overflow-x-hidden">
    <div class="overflow-x-auto">
      <div class="mx-1 w-192">
        <template v-if="projectId">
          <template v-if="isTenantProject">
            <!-- tenant mode project -->
            <NTabs v-model:value="state.alterType">
              <NTabPane
                :tab="$t('alter-schema.alter-multiple-db')"
                name="MULTI_DB"
              >
                <ProjectTenantView
                  :state="state"
                  :database-list="databaseList"
                  :environment-list="environmentList"
                  :project="state.project"
                  @dismiss="cancel"
                />
              </NTabPane>
              <NTabPane
                :tab="$t('alter-schema.alter-single-db')"
                name="SINGLE_DB"
              >
                <!-- a simple table -->
                <DatabaseTable
                  mode="PROJECT_SHORT"
                  :bordered="true"
                  :custom-click="true"
                  :database-list="databaseList"
                  @select-database="selectDatabase"
                />
              </NTabPane>
              <template #suffix>
                <BBTableSearch
                  v-if="state.alterType === 'SINGLE_DB'"
                  class="m-px"
                  :placeholder="$t('database.search-database-name')"
                  @change-text="(text: string) => (state.searchText = text)"
                />
              </template>
            </NTabs>
          </template>
          <template v-else>
            <!-- standard mode project, single/multiple databases ui -->
            <ProjectStandardView
              :state="state"
              :project="state.project"
              :database-list="databaseList"
              :environment-list="environmentList"
              @select-database="selectDatabase"
            >
              <template #header>
                <div class="flex items-center justify-end my-2">
                  <BBTableSearch
                    class="m-px"
                    :placeholder="$t('database.search-database-name')"
                    @change-text="(text: string) => (state.searchText = text)"
                  />
                </div>
              </template>
            </ProjectStandardView>
          </template>
        </template>
        <template v-else>
          <aside class="flex justify-end mb-4">
            <BBTableSearch
              class="m-px"
              :placeholder="$t('database.search-database-name')"
              @change-text="(text: string) => (state.searchText = text)"
            />
          </aside>
          <!-- a simple table -->
          <DatabaseTable
            mode="ALL_SHORT"
            :bordered="true"
            :custom-click="true"
            :database-list="databaseList"
            @select-database="selectDatabase"
          />
        </template>
      </div>
    </div>

    <!-- Create button group -->
    <div
      class="pt-4 border-t border-block-border flex items-center justify-between"
    >
      <div>
        <div
          v-if="flattenSelectedDatabaseIdList.length > 0"
          class="textinfolabel"
        >
          {{
            $t("database.selected-n-databases", {
              n: flattenSelectedDatabaseIdList.length,
            })
          }}
        </div>
      </div>

      <div class="flex items-center justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="cancel"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          v-if="showGenerateMultiDb"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          :disabled="!allowGenerateMultiDb"
          @click.prevent="generateMultiDb"
        >
          {{ $t("common.next") }}
        </button>

        <button
          v-if="showGenerateTenant"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          :disabled="!allowGenerateTenant"
          @click.prevent="generateTenant"
        >
          {{ $t("common.next") }}
        </button>
      </div>
    </div>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.multi-tenancy"
    @cancel="state.showFeatureModal = false"
  />

  <GhostDialog ref="ghostDialog" />
</template>

<script lang="ts">
import { computed, reactive, PropType, defineComponent, ref } from "vue";
import { useRouter } from "vue-router";
import { NTabs, NTabPane } from "naive-ui";
import { useEventListener } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import DatabaseTable from "../DatabaseTable.vue";
import {
  baseDirectoryWebUrl,
  Database,
  DatabaseId,
  Project,
  ProjectId,
  Repository,
  UNKNOWN_ID,
} from "@/types";
import { allowGhostMigration, sortDatabaseList } from "@/utils";
import ProjectStandardView, {
  State as ProjectStandardState,
} from "./ProjectStandardView.vue";
import ProjectTenantView, {
  State as ProjectTenantState,
} from "./ProjectTenantView.vue";
import { State as CommonTenantState } from "./CommonTenantView.vue";
import GhostDialog from "./GhostDialog.vue";
import {
  hasFeature,
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentList,
  useProjectStore,
  useRepositoryStore,
} from "@/store";
import dayjs from "dayjs";

type LocalState = ProjectStandardState &
  ProjectTenantState &
  CommonTenantState & {
    project?: Project;
    showFeatureModal: boolean;
    searchText: string;
  };

export default defineComponent({
  name: "AlterSchemaPrepForm",
  components: {
    DatabaseTable,
    ProjectStandardView,
    ProjectTenantView,
    NTabs,
    NTabPane,
    GhostDialog,
  },
  props: {
    projectId: {
      type: Number as PropType<ProjectId>,
      default: undefined,
    },
    type: {
      type: String as PropType<
        "bb.issue.database.schema.update" | "bb.issue.database.data.update"
      >,
      required: true,
    },
  },
  emits: ["dismiss"],
  setup(props, { emit }) {
    const router = useRouter();

    const currentUser = useCurrentUser();
    const projectStore = useProjectStore();
    const repositoryStore = useRepositoryStore();

    const ghostDialog = ref<InstanceType<typeof GhostDialog>>();

    useEventListener(window, "keydown", (e) => {
      if (e.code === "Escape") {
        cancel();
      }
    });

    const state = reactive<LocalState>({
      project: props.projectId
        ? projectStore.getProjectById(props.projectId)
        : undefined,
      alterType: "SINGLE_DB",
      selectedDatabaseIdListForEnvironment: new Map(),
      tenantProjectId: undefined,
      selectedDatabaseName: undefined,
      deployingTenantDatabaseList: [],
      showFeatureModal: false,
      searchText: "",
    });

    // Returns true if alter schema, false if change data.
    const isAlterSchema = computed((): boolean => {
      return props.type === "bb.issue.database.schema.update";
    });

    const isTenantProject = computed((): boolean => {
      return state.project?.tenantMode === "TENANT";
    });

    if (isTenantProject.value) {
      // For tenant mode projects, alter multiple db via DeploymentConfig
      // is the default suggested way.
      state.alterType = "MULTI_DB";
    }

    const environmentList = useEnvironmentList(["NORMAL"]);

    const databaseList = computed(() => {
      const databaseStore = useDatabaseStore();
      let list;
      if (props.projectId) {
        list = databaseStore.getDatabaseListByProjectId(props.projectId);
      } else {
        list = databaseStore.getDatabaseListByPrincipalId(currentUser.value.id);
      }

      const keyword = state.searchText.trim();
      if (keyword) {
        list = list.filter((db) => db.name.toLowerCase().includes(keyword));
      }

      return sortDatabaseList(cloneDeep(list), environmentList.value);
    });

    const flattenSelectedDatabaseIdList = computed(() => {
      const flattenDatabaseIdList: DatabaseId[] = [];
      for (const databaseIdList of state.selectedDatabaseIdListForEnvironment.values()) {
        flattenDatabaseIdList.push(...databaseIdList);
      }
      return flattenDatabaseIdList;
    });

    const showGenerateMultiDb = computed(() => {
      if (isTenantProject.value) return false;
      return state.alterType === "MULTI_DB";
    });

    const allowGenerateMultiDb = computed(() => {
      return flattenSelectedDatabaseIdList.value.length > 0;
    });

    // 'normal' -> normal migration
    // 'online' -> online migration
    // false -> user clicked cancel button
    const isUsingGhostMigration = async (databaseList: Database[]) => {
      // Gh-ost is not available for tenant mode yet.
      if (databaseList.some((db) => db.project.tenantMode === "TENANT")) {
        return "normal";
      }

      // never available for "bb.issue.database.data.update"
      if (props.type === "bb.issue.database.data.update") {
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

    // Also works when single db selected.
    const generateMultiDb = async () => {
      const selectedDatabaseIdList = [...flattenSelectedDatabaseIdList.value];

      const selectedDatabaseList = selectedDatabaseIdList.map(
        (id) => databaseList.value.find((db) => db.id === id)!
      );

      const mode = await isUsingGhostMigration(selectedDatabaseList);
      if (mode === false) {
        return;
      }

      const query: Record<string, any> = {
        template: props.type,
        name: generateIssueName(
          selectedDatabaseList.map((db) => db.name),
          mode === "online"
        ),
        project: props.projectId,
        // The server-side will sort the databases by environment.
        // So we need not to sort them here.
        databaseList: selectedDatabaseIdList.join(","),
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

    const showGenerateTenant = computed(() => {
      // True when a tenant project is selected and "MULTI_DB" is selected.
      if (isTenantProject.value && state.alterType === "MULTI_DB") {
        return true;
      }
      return false;
    });

    const allowGenerateTenant = computed(() => {
      if (!state.selectedDatabaseName) return false;

      // not allowed when database list filtered by deployment config is empty
      // which means no database will be deployed
      if (state.deployingTenantDatabaseList.length === 0) return false;

      return true;
    });

    const generateTenant = async () => {
      if (!hasFeature("bb.feature.multi-tenancy")) {
        state.showFeatureModal = true;
        return;
      }

      emit("dismiss");

      const projectId = props.projectId || state.tenantProjectId;
      if (!projectId) return;

      const project = projectStore.getProjectById(projectId) as Project;

      if (project.id === UNKNOWN_ID) return;

      if (project.workflowType === "UI") {
        router.push({
          name: "workspace.issue.detail",
          params: {
            issueSlug: "new",
          },
          query: {
            template: props.type,
            name: generateIssueName([state.selectedDatabaseName!], false),
            project: project.id,
            databaseName: state.selectedDatabaseName,
            mode: "tenant",
          },
        });
      } else if (project.workflowType === "VCS") {
        repositoryStore
          .fetchRepositoryByProjectId(project.id)
          .then((repository: Repository) => {
            window.open(
              baseDirectoryWebUrl(repository, {
                DB_NAME: state.selectedDatabaseName!,
                TYPE:
                  props.type === "bb.issue.database.schema.update"
                    ? "migrate"
                    : "data",
              }),
              "_blank"
            );
          });
      }
    };

    const selectDatabase = async (database: Database) => {
      if (database.project.workflowType == "UI") {
        const mode = await isUsingGhostMigration([database]);
        if (mode === false) {
          return;
        }
        emit("dismiss");

        const query: Record<string, any> = {
          template: props.type,
          name: generateIssueName([database.name], mode === "online"),
          project: database.project.id,
          databaseList: database.id,
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
      } else if (database.project.workflowType == "VCS") {
        repositoryStore
          .fetchRepositoryByProjectId(database.project.id)
          .then((repository: Repository) => {
            window.open(
              baseDirectoryWebUrl(repository, {
                DB_NAME: database.name,
                ENV_NAME: database.instance.environment.name,
                TYPE:
                  props.type === "bb.issue.database.schema.update"
                    ? "migrate"
                    : "data",
              }),
              "_blank"
            );
          });
        emit("dismiss");
      }
    };

    const cancel = () => {
      emit("dismiss");
    };

    const generateIssueName = (
      databaseNameList: string[],
      isOnlineMode: boolean
    ) => {
      // Create a user friendly default issue name
      const issueNameParts: string[] = [];
      if (databaseNameList.length === 1) {
        issueNameParts.push(`[${databaseNameList[0]}]`);
      } else {
        issueNameParts.push(`[${databaseNameList.length} databases]`);
      }
      if (isOnlineMode) {
        issueNameParts.push("Online schema change");
      } else {
        issueNameParts.push(
          isAlterSchema.value ? `Alter schema` : `Change data`
        );
      }
      issueNameParts.push(dayjs().format("@MM-DD HH:mm"));

      return issueNameParts.join(" ");
    };

    return {
      state,
      ghostDialog,
      isAlterSchema,
      isTenantProject,
      environmentList,
      databaseList,
      showGenerateMultiDb,
      allowGenerateMultiDb,
      flattenSelectedDatabaseIdList,
      generateMultiDb,
      showGenerateTenant,
      allowGenerateTenant,
      generateTenant,
      selectDatabase,
      cancel,
    };
  },
});
</script>
