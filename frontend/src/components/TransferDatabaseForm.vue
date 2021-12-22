<template>
  <div class="px-4 space-y-6 w-208">
    <div v-if="projectId != DEFAULT_PROJECT_ID" class="textlabel">
      <div v-if="state.transferSource == 'DEFAULT'" class="textinfolabel mb-2">
        {{ $t("quick-action.transfer-in-db-hint") }}
      </div>
      <div class="radio-set-row">
        <div class="flex flex-row">
          <div class="radio">
            <input
              v-model="state.transferSource"
              tabindex="-1"
              type="radio"
              class="btn"
              value="DEFAULT"
            />
            <label class="label">
              {{ $t("quick-action.from-default-project") }}
            </label>
          </div>
        </div>
        <div class="radio">
          <input
            v-model="state.transferSource"
            tabindex="-1"
            type="radio"
            class="btn"
            value="OTHER"
          />
          <label class="label">
            {{ $t("quick-action.from-other-projects") }}
          </label>
        </div>
      </div>
    </div>

    <DatabaseTable
      :mode="'ALL_SHORT'"
      :bordered="true"
      :custom-click="true"
      :database-list="databaseList"
      @select-database="selectDatabase"
    />

    <!-- Update button group -->
    <div class="pt-4 border-t border-block-border flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        {{ $t("common.cancel") }}
      </button>
    </div>

    <BBAlert
      v-if="state.showModal"
      :style="'INFO'"
      :ok-text="$t('common.transfer')"
      :title="
        $t('quick-action.transfer-in-db-alert', {
          dbName: selectedDatabaseName,
        })
      "
      @ok="
        () => {
          state.showModal = false;
          transferDatabase();
        }
      "
      @cancel="state.showModal = false"
    >
    </BBAlert>
  </div>
</template>

<script lang="ts">
import { computed, PropType, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import { cloneDeep } from "lodash-es";
import DatabaseTable from "../components/DatabaseTable.vue";
import { Database, ProjectId, DEFAULT_PROJECT_ID } from "../types";
import { sortDatabaseList } from "../utils";

type TransferSource = "DEFAULT" | "OTHER";

interface LocalState {
  selectedDatabase?: Database;
  transferSource: TransferSource;
  showModal: boolean;
}

export default {
  name: "TransferDatabaseForm",
  components: {
    DatabaseTable,
  },
  props: {
    projectId: {
      required: true,
      type: Number as PropType<ProjectId>,
    },
  },
  emits: ["submit", "dismiss"],
  setup(props, { emit }) {
    const store = useStore();

    const state = reactive<LocalState>({
      transferSource:
        props.projectId == DEFAULT_PROJECT_ID ? "OTHER" : "DEFAULT",
      showModal: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareDatabaseListForDefaultProject = () => {
      store.dispatch(
        "database/fetchDatabaseListByProjectId",
        DEFAULT_PROJECT_ID
      );
    };

    watchEffect(prepareDatabaseListForDefaultProject);

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"](["NORMAL"]);
    });

    const databaseList = computed(() => {
      var list;
      if (state.transferSource == "DEFAULT") {
        list = cloneDeep(
          store.getters["database/databaseListByProjectId"](DEFAULT_PROJECT_ID)
        );
      } else {
        list = cloneDeep(
          store.getters["database/databaseListByPrincipalId"](
            currentUser.value.id
          )
        ).filter((item: Database) => item.project.id != props.projectId);
      }

      return sortDatabaseList(list, environmentList.value);
    });

    const selectedDatabaseName = computed(() => {
      return state.selectedDatabase?.name;
    });

    const selectDatabase = (database: Database) => {
      state.selectedDatabase = database;
      state.showModal = true;
    };

    const transferDatabase = () => {
      store
        .dispatch("database/transferProject", {
          databaseId: state.selectedDatabase!.id,
          projectId: props.projectId,
        })
        .then((updatedDatabase) => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully transferred '${updatedDatabase.name}' to project '${updatedDatabase.project.name}'.`,
          });
          emit("dismiss");
        });
    };

    const cancel = () => {
      emit("dismiss");
    };

    return {
      DEFAULT_PROJECT_ID,
      state,
      databaseList,
      selectedDatabaseName,
      selectDatabase,
      transferDatabase,
      cancel,
    };
  },
};
</script>
