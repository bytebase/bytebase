<template>
  <form class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-2">
      <template v-if="state.singleEnvironment">
        <div class="col-span-2 col-start-1 w-64">
          <label for="environment" class="textlabel">
            Environment <span style="color: red">*</span>
          </label>
          <EnvironmentSelect
            class="mt-1 w-full"
            id="environment"
            name="environment"
            :selectedId="state.singleStageConfig.environmentId"
            @select-environment-id="
              (environmentId) => {
                state.singleStageConfig.environmentId = environmentId;
              }
            "
          />
        </div>

        <!-- If project is supplied, we put ProjectSelect before DatabaseSelect,
             Otherwise, we put it after. This is based on the assumption that
             it's relatively easy to directly select the database instead of selecting
             project and then selecting database in that project -->
        <div v-if="projectId != UNKNOWN_ID" class="col-span-2 col-start-1 w-64">
          <label for="project" class="textlabel">
            Project <span style="color: red">*</span>
          </label>
          <!-- Disable the selection if having preset project -->
          <ProjectSelect
            class="mt-1"
            id="project"
            name="project"
            :disabled="true"
            :selectedId="state.project.id"
            @select-project-id="
              (projectId) => {
                changeProjectId(projectId);
              }
            "
          />
        </div>

        <div class="col-span-2 col-start-1 w-64">
          <label for="database" class="textlabel">
            Database <span style="color: red">*</span>
          </label>
          <DatabaseSelect
            class="mt-1 w-full"
            id="database"
            name="database"
            :selectedId="state.singleStageConfig.databaseId"
            :environmentId="state.singleStageConfig.environmentId"
            :projectId="state.project.id"
            :mode="'USER'"
            @select-database-id="
              (databaseId) => {
                changeDatabaseId(databaseId);
              }
            "
          />
        </div>

        <div v-if="projectId == UNKNOWN_ID" class="col-span-2 col-start-1 w-64">
          <label for="project" class="textlabel">
            Project <span style="color: red">*</span>
          </label>
          <!-- Disable the selection if having preset project -->
          <ProjectSelect
            class="mt-1"
            id="project"
            name="project"
            :selectedId="state.project.id"
            @select-project-id="
              (projectId) => {
                changeProjectId(projectId);
              }
            "
          />
        </div>
      </template>
    </div>
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        Cancel
      </button>
      <button
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowNext"
        @click.prevent="goNext"
      >
        Next
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, reactive, onMounted, onUnmounted, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import ProjectSelect from "../components/ProjectSelect.vue";
import DatabaseSelect from "../components/DatabaseSelect.vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import {
  Database,
  DatabaseId,
  EnvironmentId,
  Project,
  ProjectId,
  UNKNOWN_ID,
} from "../types";

type StageConfig = {
  environmentId: EnvironmentId;
  databaseId: DatabaseId;
};

interface LocalState {
  project: Project;
  singleEnvironment: boolean;
  singleStageConfig: StageConfig;
  stageConfigList: StageConfig[];
}

export default {
  name: "AlterSchemaPrepForm",
  emits: ["dismiss"],
  props: {
    projectId: {
      required: true,
      type: Number as PropType<ProjectId>,
    },
  },
  components: { ProjectSelect, DatabaseSelect, EnvironmentSelect },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const keyboardHandler = (e: KeyboardEvent) => {
      if (e.code == "Escape") {
        cancel();
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    const state = reactive<LocalState>({
      project: store.getters["project/projectById"](props.projectId),
      singleEnvironment: true,
      singleStageConfig: {
        environmentId: UNKNOWN_ID,
        databaseId: UNKNOWN_ID,
      },
      stageConfigList: [],
    });

    const allowNext = computed(() => {
      if (state.singleEnvironment) {
        return (
          state.project.id != UNKNOWN_ID &&
          state.singleStageConfig.environmentId != UNKNOWN_ID &&
          state.singleStageConfig.databaseId != UNKNOWN_ID
        );
      }
      return true;
    });

    const changeProjectId = (projectId: ProjectId) => {
      state.project = store.getters["project/projectById"](projectId);
      if (state.singleEnvironment) {
      } else {
        state.stageConfigList = [];
      }
    };

    const changeDatabaseId = (databaseId: DatabaseId) => {
      state.singleStageConfig.databaseId = databaseId;

      if (databaseId != UNKNOWN_ID) {
        const database: Database = store.getters["database/databaseById"](
          state.singleStageConfig.databaseId
        );
        state.project = database.project;
      }
    };

    const cancel = () => {
      emit("dismiss");
    };

    const goNext = () => {
      emit("dismiss");

      if (state.singleEnvironment) {
        const database = store.getters["database/databaseById"](
          state.singleStageConfig.databaseId
        );

        router.push({
          name: "workspace.issue.detail",
          params: {
            issueSlug: "new",
          },
          query: {
            template: "bb.issue.db.schema.update",
            name: `[${database.name}] Alter schema`,
            project: state.project.id,
            databaseList: database.id,
          },
        });
      }
    };

    return {
      UNKNOWN_ID,
      state,
      allowNext,
      changeProjectId,
      changeDatabaseId,
      cancel,
      goNext,
    };
  },
};
</script>
