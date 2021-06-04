<template>
  <div class="mx-4 space-y-6 w-160 divide-y divide-block-border">
    <DatabaseTable
      :mode="projectId == UNKNOWN_ID ? 'ALL_SHORT' : 'PROJECT_SHORT'"
      :bordered="true"
      :customClick="true"
      :databaseList="databaseList"
      @select-database-id="selectDatabaseId"
    />
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        Cancel
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive, onMounted, onUnmounted, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DatabaseTable from "../components/DatabaseTable.vue";
import { Database, DatabaseId, Project, ProjectId, UNKNOWN_ID } from "../types";

interface LocalState {
  project: Project;
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
  components: {
    DatabaseTable,
  },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

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
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]([
        "NORMAL",
        "ARCHIVED",
      ]);
    });

    const databaseList = computed(() => {
      var list;
      if (props.projectId == UNKNOWN_ID) {
        list = store.getters["database/databaseListByPrincipalId"](
          currentUser.value.id
        );
      } else {
        list = store.getters["database/databaseListByProjectId"](
          props.projectId
        );
      }

      // Sort the list to put prod items first.
      return list.sort((a: Database, b: Database) => {
        var aEnvIndex = -1;
        var bEnvIndex = -1;

        for (var i = 0; i < environmentList.value.length; i++) {
          if (environmentList.value[i].id == a.instance.environment.id) {
            aEnvIndex = i;
          }
          if (environmentList.value[i].id == b.instance.environment.id) {
            bEnvIndex = i;
          }

          if (aEnvIndex != -1 && bEnvIndex != -1) {
            break;
          }
        }
        return bEnvIndex - aEnvIndex;
      });
    });

    const selectDatabaseId = (databaseId: DatabaseId) => {
      emit("dismiss");

      const database = store.getters["database/databaseById"](databaseId);

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
    };

    const cancel = () => {
      emit("dismiss");
    };

    return {
      UNKNOWN_ID,
      state,
      databaseList,
      selectDatabaseId,
      cancel,
    };
  },
};
</script>
