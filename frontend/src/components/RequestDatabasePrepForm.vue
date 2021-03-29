<template>
  <form class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-4">
      <div class="col-span-2 col-start-2 w-64">
        <label for="environment" class="textlabel">
          Environment <span style="color: red">*</span>
        </label>
        <EnvironmentSelect
          class="mt-1"
          id="environment"
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
        <div class="col-span-2 col-start-2 w-64 space-y-2">
          <div class="hidden sm:flex sm:flex-row radio-set justify-between">
            <div class="radio">
              <input
                name="Create new"
                tabindex="-1"
                type="radio"
                class="btn"
                value="ON"
                v-model="state.create"
              />
              <label class="label whitespace-nowrap">Create new</label>
            </div>
            <div class="radio">
              <input
                name="Access existing"
                tabindex="-1"
                type="radio"
                class="btn"
                value="OFF"
                v-model="state.create"
              />
              <label class="label whitespace-nowrap">Access existing DB</label>
            </div>
          </div>

          <div class="space-y-1">
            <label for="database" class="textlabel">
              Database name <span style="color: red">*</span>
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
                :value="state.readOnly"
                @toggle="
                  (on) => {
                    state.readOnly = on;
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
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
import DatabaseSelect from "../components/DatabaseSelect.vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import { EnvironmentId } from "../types";

interface LocalState {
  environmentId?: EnvironmentId;
  // Radio button only accept string value
  create: "ON" | "OFF";
  databaseName?: string;
  databaseId?: string;
  readonly: boolean;
}

export default {
  name: "RequestDatabasePrepForm",
  emits: ["dismiss"],
  props: {},
  components: { DatabaseSelect, EnvironmentSelect },
  setup(props, { emit }) {
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
      create: "ON",
      readonly: true,
    });

    const allowRequest = computed(() => {
      return (
        state.environmentId &&
        ((state.create == "ON" && !isEmpty(state.databaseName)) ||
          (state.create == "OFF" && state.databaseId))
      );
    });

    const cancel = () => {
      emit("dismiss");
    };

    const request = () => {
      emit("dismiss");
      if (state.create == "ON") {
        router.push({
          name: "workspace.task.detail",
          params: {
            taskSlug: "new",
          },
          query: {
            template: "bytebase.database.create",
            environment: state.environmentId,
            database: state.databaseName,
          },
        });
      }
    };

    return {
      state,
      allowRequest,
      cancel,
      request,
    };
  },
};
</script>
