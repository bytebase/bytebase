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
          :selectedId="state.projectId"
          @select-project-id="
            (projectId) => {
              state.projectId = projectId;
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
          :selectedId="state.environmentId"
          @select-environment-id="
            (environmentId) => {
              state.environmentId = environmentId;
            }
          "
        />
      </div>

      <template v-if="state.environmentId">
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
                :selectedId="state.databaseId"
                :mode="'ENVIRONMENT'"
                :environmentId="state.environmentId"
                @select-database-id="
                  (databaseId) => {
                    state.databaseId = databaseId;
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
import {
  computed,
  reactive,
  onMounted,
  onUnmounted,
  defineComponent,
} from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
import ProjectSelect from "../components/ProjectSelect.vue";
import DatabaseSelect from "../components/DatabaseSelect.vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import { DatabaseId, EnvironmentId, ProjectId, UNKNOWN_ID } from "../types";
import { allowDatabaseAccess } from "../utils";
import { useCurrentUser, useDatabaseStore, useEnvironmentStore } from "@/store";

interface LocalState {
  environmentId: EnvironmentId;
  projectId: ProjectId;
  // Radio button only accept string value
  create: "ON" | "OFF";
  databaseName: string;
  databaseId: DatabaseId;
  readonly: boolean;
}

export default defineComponent({
  name: "RequestDatabasePrepForm",
  components: { ProjectSelect, DatabaseSelect, EnvironmentSelect },
  props: {},
  emits: ["dismiss"],
  setup(props, { emit }) {
    const store = useStore();
    const databaseStore = useDatabaseStore();
    const router = useRouter();

    const currentUser = useCurrentUser();

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
      environmentId: UNKNOWN_ID,
      projectId: UNKNOWN_ID,
      create: "ON",
      databaseName: "",
      databaseId: UNKNOWN_ID,
      readonly: true,
    });

    const alreadyGranted = computed(() => {
      if (state.create == "ON") {
        return false;
      }

      if (!state.databaseId) {
        return false;
      }

      const database = databaseStore.getDatabaseById(state.databaseId);
      return allowDatabaseAccess(
        database,
        currentUser.value,
        state.readonly ? "RO" : "RW"
      );
    });

    const allowRequest = computed(() => {
      return (
        state.environmentId != UNKNOWN_ID &&
        state.projectId != UNKNOWN_ID &&
        ((state.create == "ON" && !isEmpty(state.databaseName)) ||
          (state.create == "OFF" && state.databaseId != UNKNOWN_ID)) &&
        !alreadyGranted.value
      );
    });

    const cancel = () => {
      emit("dismiss");
    };

    const request = () => {
      emit("dismiss");

      const environment = useEnvironmentStore().getEnvironmentById(
        state.environmentId
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
            environment: state.environmentId,
            project: state.projectId,
            databaseName: state.databaseName,
          },
        });
      } else {
        const database = databaseStore.getDatabaseById(state.databaseId);
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
            environment: state.environmentId,
            project: state.projectId,
            databaseList: state.databaseId,
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
});
</script>
