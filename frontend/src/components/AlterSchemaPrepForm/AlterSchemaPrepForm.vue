<template>
  <div class="mx-4 space-y-4 max-w-min overflow-x-hidden">
    <VCSTipsInfo :project="state.project" />

    <div class="overflow-x-auto">
      <div class="mx-1" :class="wrapperClass">
        <template v-if="projectId">
          <template v-if="isTenantProject">
            <!-- tenant mode project -->
            <ProjectTenantView
              :state="state"
              :database-list="databaseList"
              :environment-list="environmentList"
              :project="state.project"
              @dismiss="cancel"
            />
          </template>
          <template v-else>
            <!-- standard mode project, single/multiple databases ui -->
            <ProjectStandardView
              :state="state"
              :project="state.project"
              :database-list="databaseList"
              :environment-list="environmentList"
              @select-database="selectDatabase"
            />
          </template>
        </template>
        <template v-else>
          <NTabs v-model:value="state.tab" type="line">
            <NTabPane :tab="$t('project.mode.standard')" name="standard">
              <!-- a simple table -->
              <DatabaseTable
                mode="ALL_SHORT"
                :bordered="true"
                :custom-click="true"
                :database-list="standardProjectDatabaseList"
                @select-database="selectDatabase"
              />
            </NTabPane>
            <NTabPane :tab="$t('project.mode.tenant')" name="tenant">
              <CommonTenantView
                :state="state"
                :database-list="databaseList"
                :environment-list="environmentList"
                @dismiss="cancel"
              />
            </NTabPane>
          </NTabs>
        </template>
      </div>
    </div>

    <!-- Create button group -->
    <div class="pt-4 border-t border-block-border flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        v-if="state.alterType == 'MULTI_DB'"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowGenerateMultiDb"
        @click.prevent="generateMultiDb"
      >
        {{ $t("common.next") }}
      </button>

      <button
        v-if="isTenantProject || (!projectId && state.tab === 'tenant')"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowGenerateTenant"
        @click.prevent="generateTenant"
      >
        {{ $t("common.next") }}
      </button>
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
import DatabaseTable from "../DatabaseTable.vue";
import {
  baseDirectoryWebUrl,
  Database,
  DatabaseId,
  Project,
  ProjectId,
  Repository,
  UNKNOWN_ID,
} from "../../types";
import { allowGhostMigration, isDev, sortDatabaseList } from "../../utils";
import { cloneDeep } from "lodash-es";
import VCSTipsInfo from "./VCSTipsInfo.vue";
import ProjectStandardView, {
  State as ProjectStandardState,
} from "./ProjectStandardView.vue";
import ProjectTenantView, {
  State as ProjectTenantState,
} from "./ProjectTenantView.vue";
import CommonTenantView, {
  State as CommonTenantState,
} from "./CommonTenantView.vue";
import GhostDialog from "./GhostDialog.vue";
import { NTabs, NTabPane } from "naive-ui";
import { useEventListener } from "@vueuse/core";
import {
  hasFeature,
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentList,
  useProjectStore,
  useRepositoryStore,
} from "@/store";

type LocalState = ProjectStandardState &
  ProjectTenantState &
  CommonTenantState & {
    project?: Project;
    tab: "standard" | "tenant";
    showFeatureModal: boolean;
  };

export default defineComponent({
  name: "AlterSchemaPrepForm",
  components: {
    VCSTipsInfo,
    DatabaseTable,
    ProjectStandardView,
    ProjectTenantView,
    CommonTenantView,
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
      tab: "standard",
      alterType: "SINGLE_DB",
      selectedDatabaseIdForEnvironment: new Map(),
      tenantProjectId: undefined,
      selectedDatabaseName: undefined,
      deployingTenantDatabaseList: [],
      showFeatureModal: false,
    });

    // Returns true if alter schema, false if change data.
    const isAlterSchema = computed((): boolean => {
      return props.type === "bb.issue.database.schema.update";
    });

    const isTenantProject = computed((): boolean => {
      return state.project?.tenantMode === "TENANT";
    });

    const environmentList = useEnvironmentList(["NORMAL"]);

    const databaseList = computed(() => {
      const databaseStore = useDatabaseStore();
      var list;
      if (props.projectId) {
        list = databaseStore.getDatabaseListByProjectId(props.projectId);
      } else {
        list = databaseStore.getDatabaseListByPrincipalId(currentUser.value.id);
      }

      return sortDatabaseList(cloneDeep(list), environmentList.value);
    });

    const standardProjectDatabaseList = computed(() => {
      return databaseList.value.filter(
        (db) => db.project.tenantMode !== "TENANT"
      );
    });

    const tenantProjectDatabaseList = computed(() => {
      return databaseList.value.filter(
        (db) => db.project.tenantMode === "TENANT"
      );
    });

    const allowGenerateMultiDb = computed(() => {
      return state.selectedDatabaseIdForEnvironment.size > 0;
    });

    // 'normal' -> normal migration
    // 'online' -> online migration
    // false -> user clicked cancel button
    const isUsingGhostMigration = async (databaseList: Database[]) => {
      if (!isDev()) {
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

    const generateMultiDb = async () => {
      const databaseIdList: DatabaseId[] = [];
      const selectedDatabaseList: Database[] = [];
      for (var i = 0; i < environmentList.value.length; i++) {
        const envId = environmentList.value[i].id;
        const databaseId = state.selectedDatabaseIdForEnvironment.get(envId);
        if (databaseId) {
          databaseIdList.push(databaseId);
          selectedDatabaseList.push(
            databaseList.value.find((db) => db.id === databaseId)!
          );
        }
      }

      const mode = await isUsingGhostMigration(selectedDatabaseList);
      if (mode === false) {
        return;
      }
      const query: Record<string, any> = {
        template: props.type,
        name: isAlterSchema.value ? `Alter schema` : `Change data`,
        project: props.projectId,
        databaseList: databaseIdList.join(","),
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
            name: `[${state.selectedDatabaseName}] ${
              isAlterSchema.value ? `Alter schema` : `Change data`
            }`,
            project: project.id,
            databaseName: state.selectedDatabaseName,
            mode: "tenant",
          },
        });
      } else if (project.workflowType === "VCS") {
        repositoryStore
          .fetchRepositoryByProjectId(project.id)
          .then((repository: Repository) => {
            window.open(baseDirectoryWebUrl(repository), "_blank");
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
          name: `[${database.name}] ${
            isAlterSchema.value ? `Alter schema` : `Change data`
          }`,
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
            window.open(baseDirectoryWebUrl(repository), "_blank");
          });
        emit("dismiss");
      }
    };

    const cancel = () => {
      emit("dismiss");
    };

    const wrapperClass = computed(() => {
      // provide a wider modal to tenant view
      if (props.projectId) {
        if (isTenantProject.value) return "w-192";
        else return "w-160";
      } else {
        if (state.tab === "standard") return "w-160";
        return "w-192";
      }
    });

    return {
      wrapperClass,
      state,
      ghostDialog,
      isAlterSchema,
      isTenantProject,
      environmentList,
      databaseList,
      standardProjectDatabaseList,
      tenantProjectDatabaseList,
      allowGenerateMultiDb,
      generateMultiDb,
      allowGenerateTenant,
      generateTenant,
      selectDatabase,
      cancel,
    };
  },
});
</script>
