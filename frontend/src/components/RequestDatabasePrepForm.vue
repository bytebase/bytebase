<template>
  <form class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-2">
      <div class="col-span-2 col-start-1 w-64">
        <label for="project" class="textlabel">
          Project <span style="color: red">*</span>
        </label>
        <!-- eslint-disable vue/attribute-hyphenation -->
        <ProjectSelect
          id="project"
          class="mt-1"
          name="project"
          :selectedID="state.projectID"
          @select-project-id="
            (projectID) => {
              state.projectID = projectID;
            }
          "
        />
      </div>

      <div class="col-span-2 col-start-1 w-64">
        <label for="environment" class="textlabel">
          Environment <span style="color: red">*</span>
        </label>
        <!-- eslint-disable vue/attribute-hyphenation -->
        <EnvironmentSelect
          id="environment"
          class="mt-1 w-full"
          name="environment"
          :selectedID="state.environmentID"
          @select-environment-id="
            (environmentID) => {
              state.environmentID = environmentID;
            }
          "
        />
      </div>

      <template v-if="state.environmentID">
        <div class="col-span-2 col-start-1 w-64 space-y-2">
          <div class="hidden radio-set-row justify-between">
            <div class="radio">
              <input
                v-model="state.create"
                name="Create new"
                tabindex="-1"
                type="radio"
                class="btn"
                value="ON"
              />
              <label class="label whitespace-nowrap">Create new</label>
            </div>
            <div class="radio">
              <input
                v-model="state.create"
                name="Access existing"
                tabindex="-1"
                type="radio"
                class="btn"
                value="OFF"
              />
              <label class="label whitespace-nowrap">Access existing DB</label>
            </div>
          </div>

          <div class="space-y-1">
            <label for="database" class="textlabel">
              Database name <span style="color: red">*</span>
              <span v-if="alreadyGranted" class="text-error">
                Already granted!
              </span>
            </label>
            <BBTextField
              v-if="state.create == 'ON'"
              type="text"
              class="w-full text-sm"
              :required="true"
              :value="state.databaseName"
              :placeholder="'New database name'"
              @end-editing="(text) => (state.databaseName = text)"
              @input="state.databaseName = $event.target.value"
            />
            <div v-else class="flex flex-row space-x-4">
              <!-- eslint-disable vue/attribute-hyphenation -->
              <DatabaseSelect
                :selectedID="state.databaseID"
                :mode="'ENVIRONMENT'"
                :environmentID="state.environmentID"
                @select-database-id="
                  (databaseID) => {
                    state.databaseID = databaseID;
                  }
                "
              />
              <BBSwitch
                :label="'Read only'"
                :value="state.readonly"
                @toggle="
                  (on) => {
                    state.readonly = on;
                  }
                "
              />
            </div>
          </div>
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
        :disabled="!allowRequest"
        @click.prevent="request"
      >
        Request
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, reactive, onMounted, onUnmounted } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
import ProjectSelect from "../components/ProjectSelect.vue";
import DatabaseSelect from "../components/DatabaseSelect.vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import { DatabaseID, EnvironmentID, ProjectID, UNKNOWN_ID } from "../types";
import { allowDatabaseAccess } from "../utils";

interface LocalState {
  environmentID: EnvironmentID;
  projectID: ProjectID;
  // Radio button only accept string value
  create: "ON" | "OFF";
  databaseName: string;
  databaseID: DatabaseID;
  readonly: boolean;
}

export default {
  name: "RequestDatabasePrepForm",
  components: { ProjectSelect, DatabaseSelect, EnvironmentSelect },
  props: {},
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
      environmentID: UNKNOWN_ID,
      projectID: UNKNOWN_ID,
      create: "ON",
      databaseName: "",
      databaseID: UNKNOWN_ID,
      readonly: true,
    });

    const alreadyGranted = computed(() => {
      if (state.create == "ON") {
        return false;
      }

      if (!state.databaseID) {
        return false;
      }

      return allowDatabaseAccess(
        store.getters["database/databaseByID"](state.databaseID),
        currentUser.value,
        state.readonly ? "RO" : "RW"
      );
    });

    const allowRequest = computed(() => {
      return (
        state.environmentID != UNKNOWN_ID &&
        state.projectID != UNKNOWN_ID &&
        ((state.create == "ON" && !isEmpty(state.databaseName)) ||
          (state.create == "OFF" && state.databaseID != UNKNOWN_ID)) &&
        !alreadyGranted.value
      );
    });

    const cancel = () => {
      emit("dismiss");
    };

    const request = () => {
      emit("dismiss");

      const environment = store.getters["environment/environmentByID"](
        state.environmentID
      );
      if (state.create == "ON") {
        router.push({
          name: "workspace.issue.detail",
          params: {
            issueSlug: "new",
          },
          query: {
            template: "bb.issue.database.create",
            name: `[${environment.name}] Request new database '${state.databaseName}'`,
            environment: state.environmentID,
            project: state.projectID,
            databaseName: state.databaseName,
          },
        });
      } else {
        const database = store.getters["database/databaseByID"](
          state.databaseID
        );
        router.push({
          name: "workspace.issue.detail",
          params: {
            issueSlug: "new",
          },
          query: {
            template: "bb.issue.database.grant",
            name: `[${environment.name}] Request database '${database.name}' ${
              state.readonly ? "Readonly access" : "Read & Write access"
            }`,
            environment: state.environmentID,
            project: state.projectID,
            databaseList: state.databaseID,
            readonly: state.readonly ? "true" : "false",
          },
        });
      }
    };

    return {
      state,
      alreadyGranted,
      allowRequest,
      cancel,
      request,
    };
  },
};
</script>
