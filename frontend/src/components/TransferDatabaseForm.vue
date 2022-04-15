<template>
  <div
    class="px-4 space-y-6"
    :class="!state.selectedDatabase ? 'w-208' : 'w-112'"
  >
    <template v-if="!state.selectedDatabase">
      <div v-if="projectId != DEFAULT_PROJECT_ID" class="textlabel">
        <div
          v-if="state.transferSource == 'DEFAULT'"
          class="textinfolabel mb-2"
        >
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
    </template>

    <template v-else>
      <SelectDatabaseLabel
        :database="state.selectedDatabase"
        :target-project-id="projectId"
        @next="transferDatabase"
      >
        <template #buttons="{ next, valid }">
          <div
            class="w-full pt-4 mt-6 flex justify-end border-t border-block-border"
          >
            <button
              type="button"
              class="btn-normal py-2 px-4"
              @click.prevent="state.selectedDatabase = undefined"
            >
              {{ $t("common.back") }}
            </button>
            <button
              type="button"
              class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
              :disabled="!valid"
              @click.prevent="next"
            >
              {{ $t("common.transfer") }}
            </button>
          </div>
        </template>
      </SelectDatabaseLabel>
    </template>
  </div>
</template>

<script lang="ts">
import {
  computed,
  defineComponent,
  PropType,
  reactive,
  watchEffect,
} from "vue";
import { cloneDeep } from "lodash-es";
import DatabaseTable from "../components/DatabaseTable.vue";
import { SelectDatabaseLabel } from "../components/TransferDatabaseForm";
import {
  Database,
  ProjectId,
  DEFAULT_PROJECT_ID,
  DatabaseLabel,
} from "../types";
import { sortDatabaseList } from "../utils";
import {
  pushNotification,
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentList,
} from "@/store";

type TransferSource = "DEFAULT" | "OTHER";

interface LocalState {
  selectedDatabase?: Database;
  transferSource: TransferSource;
}

export default defineComponent({
  name: "TransferDatabaseForm",
  components: {
    DatabaseTable,
    SelectDatabaseLabel,
  },
  props: {
    projectId: {
      required: true,
      type: Number as PropType<ProjectId>,
    },
  },
  emits: ["submit", "dismiss"],
  setup(props, { emit }) {
    const databaseStore = useDatabaseStore();

    const state = reactive<LocalState>({
      transferSource:
        props.projectId == DEFAULT_PROJECT_ID ? "OTHER" : "DEFAULT",
    });

    const currentUser = useCurrentUser();

    const prepareDatabaseListForDefaultProject = () => {
      databaseStore.fetchDatabaseListByProjectId(DEFAULT_PROJECT_ID);
    };

    watchEffect(prepareDatabaseListForDefaultProject);

    const environmentList = useEnvironmentList(["NORMAL"]);

    const databaseList = computed(() => {
      var list;
      if (state.transferSource == "DEFAULT") {
        list = cloneDeep(
          databaseStore.getDatabaseListByProjectId(DEFAULT_PROJECT_ID)
        );
      } else {
        list = cloneDeep(
          databaseStore.getDatabaseListByPrincipalId(currentUser.value.id)
        ).filter((item: Database) => item.project.id != props.projectId);
      }

      return sortDatabaseList(list, environmentList.value);
    });

    const selectedDatabaseName = computed(() => {
      return state.selectedDatabase?.name;
    });

    const selectDatabase = (database: Database) => {
      state.selectedDatabase = database;
    };

    const transferDatabase = (labels: DatabaseLabel[]) => {
      databaseStore
        .transferProject({
          databaseId: state.selectedDatabase!.id,
          projectId: props.projectId,
          labels,
        })
        .then((updatedDatabase) => {
          pushNotification({
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
});
</script>
