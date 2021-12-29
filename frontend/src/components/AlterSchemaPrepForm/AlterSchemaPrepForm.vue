<template>
  <div class="mx-4 space-y-4 w-160">
    <VCSTipsInfo :project="state.project" />

    <template v-if="projectId">
      <template v-if="false && isTenantProject">
        <!-- tenant mode project, disabled for now -->
        <ProjectTenantView />
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
      <!-- a simple table now -->
      <DatabaseTable
        mode="ALL_SHORT"
        :bordered="true"
        :custom-click="true"
        :database-list="databaseList"
        @select-database="selectDatabase"
      />
      <!-- but also another view for tenant mode databases later -->
    </template>

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
    </div>
  </div>
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
} from "../../types";
import { sortDatabaseList } from "../../utils";
import { cloneDeep } from "lodash-es";
import VCSTipsInfo from "./VCSTipsInfo.vue";
import {
  default as ProjectStandardView,
  State as StandardModeState,
} from "./ProjectStandardView.vue";
import { useEventListener } from "@vueuse/core";

type LocalState = StandardModeState & {
  project?: Project;
};

export default defineComponent({
  name: "AlterSchemaPrepForm",
  components: {
    VCSTipsInfo,
    DatabaseTable,
    ProjectStandardView,
  },
  props: {
    projectId: {
      type: Number as PropType<ProjectId>,
      default: undefined,
    },
  },
  emits: ["dismiss"],
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    useEventListener(window, "keydown", (e) => {
      if (e.code === "Escape") {
        cancel();
      }
    });

    const state = reactive<LocalState>({
      project: props.projectId
        ? store.getters["project/projectById"](props.projectId)
        : undefined,
      alterType: "SINGLE_DB",
      selectedDatabaseIdForEnvironment: new Map(),
    });

    const isTenantProject = computed((): boolean => {
      return state.project?.tenantMode === "TENANT";
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"](["NORMAL"]);
    });

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
          template: "bb.issue.database.schema.update",
          name: `Alter schema`,
          project: props.projectId,
          databaseList: databaseIdList.join(","),
        },
      });
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
            template: "bb.issue.database.schema.update",
            name: `[${database.name}] Alter schema`,
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

    return {
      state,
      isTenantProject,
      environmentList,
      databaseList,
      allowGenerateMultiDb,
      generateMultDb,
      selectDatabase,
      cancel,
    };
  },
});
</script>
