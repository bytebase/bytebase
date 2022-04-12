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
        @click.prevent="generateMultDb"
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
</template>

<script lang="ts">
import { computed, reactive, PropType, defineComponent } from "vue";
import { useStore } from "vuex";
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
import { sortDatabaseList } from "../../utils";
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
import { NTabs, NTabPane } from "naive-ui";
import { useEventListener } from "@vueuse/core";
import {
  hasFeature,
  useCurrentUser,
  useEnvironmentList,
  useProjectStore,
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
    const store = useStore();
    const router = useRouter();

    const currentUser = useCurrentUser();
    const projectStore = useProjectStore();

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
      var list;
      if (props.projectId) {
        list = store.getters["database/databaseListByProjectId"](
          props.projectId
        );
      } else {
        list = store.getters["database/databaseListByPrincipalId"](
          currentUser.value.id
        );
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

    const generateMultDb = () => {
      const databaseIdList: DatabaseId[] = [];
      for (var i = 0; i < environmentList.value.length; i++) {
        if (
          state.selectedDatabaseIdForEnvironment.get(
            environmentList.value[i].id
          )
        ) {
          databaseIdList.push(
            state.selectedDatabaseIdForEnvironment.get(
              environmentList.value[i].id
            )!
          );
        }
      }
      router.push({
        name: "workspace.issue.detail",
        params: {
          issueSlug: "new",
        },
        query: {
          template: props.type,
          name: isAlterSchema.value ? `Alter schema` : `Change data`,
          project: props.projectId,
          databaseList: databaseIdList.join(","),
        },
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
        store
          .dispatch("repository/fetchRepositoryByProjectId", project.id)
          .then((repository: Repository) => {
            window.open(baseDirectoryWebUrl(repository), "_blank");
          });
      }
    };

    const selectDatabase = (database: Database) => {
      emit("dismiss");

      if (database.project.workflowType == "UI") {
        router.push({
          name: "workspace.issue.detail",
          params: {
            issueSlug: "new",
          },
          query: {
            template: props.type,
            name: `[${database.name}] ${
              isAlterSchema.value ? `Alter schema` : `Change data`
            }`,
            project: database.project.id,
            databaseList: database.id,
          },
        });
      } else if (database.project.workflowType == "VCS") {
        store
          .dispatch(
            "repository/fetchRepositoryByProjectId",
            database.project.id
          )
          .then((repository: Repository) => {
            window.open(baseDirectoryWebUrl(repository), "_blank");
          });
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
      isAlterSchema,
      isTenantProject,
      environmentList,
      databaseList,
      standardProjectDatabaseList,
      tenantProjectDatabaseList,
      allowGenerateMultiDb,
      generateMultDb,
      allowGenerateTenant,
      generateTenant,
      selectDatabase,
      cancel,
    };
  },
});
</script>
