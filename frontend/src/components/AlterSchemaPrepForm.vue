<template>
  <div class="mx-4 space-y-4 w-160">
    <template v-if="projectId">
      <div v-if="state.project.workflowType == 'VCS'" class="textlabel">
        {{ $t('alter-schema.vcs-enabled') }}
      </div>
    </template>
    <template v-else>
      <div class="flex flex-row space-x-2">
        <heroicons-outline:collection class="w-8 h-8 text-control -mt-1.5" />
        <p class="textlabel">
          {{ $t('alter-schema.vcs-info') }}
        </p>
      </div>
    </template>

    <div
      v-if="projectId && state.project.workflowType == 'UI'"
      class="mt-2 textlabel"
    >
      <div class="radio-set-row">
        <div class="radio">
          <label class="label">
            <input
              v-model="state.alterType"
              tabindex="-1"
              type="radio"
              class="btn"
              value="SINGLE_DB"
            />
           {{ $t('alter-schema.alter-single-db') }}</label>
        </div>
        <div class="radio">
          <label class="label">
            <input
              v-model="state.alterType"
              tabindex="-1"
              type="radio"
              class="btn"
              value="MULTI_DB"
            />
           {{ $t('alter-schema.alter-multiple-db') }} </label>
        </div>
      </div>
    </div>

    <template v-if="projectId && state.alterType == 'MULTI_DB'">
      <div class="textinfolabel">
        {{ $t('alter-schema.alter-multiple-db-info') }}
      </div>
      <div class="space-y-4">
        <div v-for="(environment, envIndex) in environmentList" :key="envIndex">
          <div class="mb-2">{{ environment.name }}</div>
          <div class="relative bg-white rounded-md -space-y-px">
            <template
              v-for="(database, dbIndex) in databaseList.filter(
                (item) => item.instance.environment.id == environment.id
              )"
              :key="dbIndex"
            >
              <label
                class="
                  border-control-border
                  relative
                  border
                  p-3
                  flex flex-col
                  md:pl-4 md:pr-6 md:grid md:grid-cols-2
                "
                :class="
                  database.syncStatus == 'OK'
                    ? 'cursor-pointer'
                    : 'cursor-not-allowed'
                "
              >
                <div class="radio text-sm">
                  <input
                    v-if="database.syncStatus == 'OK'"
                    type="radio"
                    class="btn"
                    :checked="
                      state.selectedDatabaseIdForEnvironment.get(
                        environment.id
                      ) == database.id
                    "
                    @change="
                      selectDatabaseIdForEnvironment(
                        database.id,
                        environment.id
                      )
                    "
                  />
                  <span
                    class="font-medium"
                    :class="
                      database.syncStatus == 'OK'
                        ? 'ml-2 text-main'
                        : 'ml-6 text-control-light'
                    "
                    >{{ database.name }}</span
                  >
                </div>
                <p
                  class="
                    textinfolabel
                    ml-6
                    pl-1
                    text-sm
                    md:ml-0 md:pl-0 md:text-right
                  "
                >
                  {{ $t('database.last-sync-status') }}:
                  <span
                    :class="
                      database.syncStatus == 'OK'
                        ? 'textlabel'
                        : 'text-sm font-medium text-error'
                    "
                    >{{ database.syncStatus }}</span
                  >
                </p>
              </label>
            </template>
            <label
              class="
                border-control-border
                relative
                border
                p-3
                flex flex-col
                cursor-pointer
                md:pl-4 md:pr-6 md:grid md:grid-cols-3
              "
            >
              <div class="radio space-x-2 text-sm">
                <input
                  type="radio"
                  class="btn"
                  :checked="
                    state.selectedDatabaseIdForEnvironment.get(environment.id)
                      ? 0
                      : 1
                  "
                  @input="clearDatabaseIdForEnvironment(environment.id)"
                />
                <span class="ml-3 font-medium text-main uppercase">{{ $t('common.skip') }}</span>
              </div>
            </label>
          </div>
        </div>
      </div>
    </template>
    <template v-else>
      <DatabaseTable
        :mode="projectId ? 'PROJECT_SHORT' : 'ALL_SHORT'"
        :bordered="true"
        :custom-click="true"
        :database-list="databaseList"
        @select-database="selectDatabase"
      />
    </template>
    <!-- Create button group -->
    <div class="pt-4 border-t border-block-border flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        {{ $t('common.cancel') }}
      </button>
      <button
        v-if="state.alterType == 'MULTI_DB'"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowGenerateMultiDb"
        @click.prevent="generateMultDb"
      >
        {{ $t('common.next') }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive, onMounted, onUnmounted, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DatabaseTable from "../components/DatabaseTable.vue";
import {
  baseDirectoryWebUrl,
  Database,
  DatabaseId,
  EnvironmentId,
  Project,
  ProjectId,
  Repository,
} from "../types";
import { sortDatabaseList } from "../utils";
import { cloneDeep } from "lodash-es";

type AlterType = "SINGLE_DB" | "MULTI_DB";

interface LocalState {
  project?: Project;
  alterType: AlterType;
  selectedDatabaseIdForEnvironment: Map<EnvironmentId, DatabaseId>;
}

export default {
  name: "AlterSchemaPrepForm",
  components: {
    DatabaseTable,
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
        : undefined,
      alterType: "SINGLE_DB",
      selectedDatabaseIdForEnvironment: new Map(),
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

    const selectDatabaseIdForEnvironment = (
      databaseId: DatabaseId,
      environmentId: EnvironmentId
    ) => {
      state.selectedDatabaseIdForEnvironment.set(environmentId, databaseId);
    };

    const clearDatabaseIdForEnvironment = (environmentId: EnvironmentId) => {
      state.selectedDatabaseIdForEnvironment.delete(environmentId);
    };

    const cancel = () => {
      emit("dismiss");
    };

    return {
      state,
      environmentList,
      databaseList,
      allowGenerateMultiDb,
      generateMultDb,
      selectDatabase,
      selectDatabaseIdForEnvironment,
      clearDatabaseIdForEnvironment,
      cancel,
    };
  },
};
</script>
