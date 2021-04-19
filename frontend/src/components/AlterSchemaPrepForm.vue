<template>
  <form class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-2">
      <div class="col-span-2 col-start-1 w-64">
        <label for="project" class="textlabel">
          Project <span style="color: red">*</span>
        </label>
        <!-- Disable the selection if having preset project -->
        <ProjectSelect
          class="mt-1"
          id="project"
          name="project"
          :disabled="projectId != null"
          :selectedId="state.project.id"
          @select-project-id="
            (projectId) => {
              changeProjectId(projectId);
            }
          "
        />
      </div>

      <template v-if="state.singleEnvironment">
        <div class="col-span-2 col-start-1 w-64">
          <label for="environment" class="textlabel">
            Environment <span style="color: red">*</span>
          </label>
          <EnvironmentSelect
            class="mt-1 w-full"
            id="environment"
            name="environment"
            :selectedId="state.singleTaskConfig.environmentId"
            @select-environment-id="
              (environmentId) => {
                state.singleTaskConfig.environmentId = environmentId;
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
            :selectedId="state.singleTaskConfig.databaseId"
            :environmentId="state.singleTaskConfig.environmentId"
            :projectId="state.project.id"
            :mode="'USER'"
            @select-database-id="
              (databaseId) => {
                state.singleTaskConfig.databaseId = databaseId;
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
  DatabaseId,
  EnvironmentId,
  Project,
  ProjectId,
  unknown,
  UNKNOWN_ID,
} from "../types";

type TaskConfig = {
  environmentId: EnvironmentId;
  databaseId: DatabaseId;
};

interface LocalState {
  project: Project;
  singleEnvironment: boolean;
  singleTaskConfig: TaskConfig;
  taskConfigList: TaskConfig[];
}

export default {
  name: "AlterSchemaPrepForm",
  emits: ["dismiss"],
  props: {
    projectId: {
      type: String as PropType<ProjectId>,
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
      project: props.projectId
        ? store.getters["project/projectById"](props.projectId)
        : (unknown("PROJECT") as Project),
      singleEnvironment: true,
      singleTaskConfig: {
        environmentId: UNKNOWN_ID,
        databaseId: UNKNOWN_ID,
      },
      taskConfigList: [],
    });

    const allowNext = computed(() => {
      if (state.singleEnvironment) {
        return (
          state.project.id != UNKNOWN_ID &&
          state.singleTaskConfig.environmentId != UNKNOWN_ID &&
          state.singleTaskConfig.databaseId != UNKNOWN_ID
        );
      }
      return true;
    });

    const changeProjectId = (projectId: ProjectId) => {
      state.project = store.getters["project/projectById"](projectId);
      if (state.singleEnvironment) {
      } else {
        state.taskConfigList = [];
      }
    };

    const cancel = () => {
      emit("dismiss");
    };

    const goNext = () => {
      emit("dismiss");

      if (state.singleEnvironment) {
        const database = store.getters["database/databaseById"](
          state.singleTaskConfig.databaseId
        );

        router.push({
          name: "workspace.issue.detail",
          params: {
            issueSlug: "new",
          },
          query: {
            template: "bytebase.database.schema.update",
            name: `[${database.name}] Alter schema`,
            databaseList: database.id,
          },
        });
      }
    };

    return {
      state,
      allowNext,
      changeProjectId,
      cancel,
      goNext,
    };
  },
};
</script>
