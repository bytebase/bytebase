<template>
  <div class="px-4 py-2 space-y-6 divide-y divide-block-border">
    <div class="grid grid-cols-1 gap-y-6 gap-x-4">
      <div class="col-span-1">
        <label for="name" class="textlabel">
          Environment Name <span class="text-red-600">*</span>
        </label>
        <BBTextField
          class="mt-2 w-full"
          :disabled="!allowEdit"
          :required="true"
          :value="state.environment.name"
          @input="state.environment.name = $event.target.value"
        />
      </div>
      <div class="col-span-1">
        <label class="textlabel"> Approval Policy </label>
        <div class="mt-1 textinfolabel">
          For updating schema on the existing database, this setting controls
          whether the task requires manual approval
          <a
            href="https://docs.bytebase.com/use-bytebase/approval-policy?ref=console"
            target="_blank"
            class="normal-link"
          >
            detailed guide</a
          >.
        </div>
        <div class="mt-4 flex flex-col space-y-4">
          <div class="flex space-x-4">
            <input
              name="manual-approval-always"
              tabindex="-1"
              type="radio"
              class="
                text-accent
                disabled:text-accent-disabled
                focus:ring-accent
              "
              value="MANUAL_APPROVAL_ALWAYS"
              :disabled="!allowEdit"
              v-model="state.environment.approvalPolicy"
            />
            <div class="-mt-0.5">
              <div class="textlabel">Require manual approval</div>
              <div class="mt-1 textinfolabel">
                Pending schema migration task will only be executed after it's
                manually approved.
              </div>
            </div>
          </div>
          <div class="flex space-x-4">
            <input
              name="manual-approval-never"
              tabindex="-1"
              type="radio"
              class="
                text-accent
                disabled:text-accent-disabled
                focus:ring-accent
              "
              value="MANUAL_APPROVAL_NEVER"
              :disabled="!allowEdit"
              v-model="state.environment.approvalPolicy"
            />
            <div class="-mt-0.5">
              <div class="textlabel">Skip manual approval</div>
              <div class="mt-1 textinfolabel">
                Pending schema migration task will be executed automatically.
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <!-- Create button group -->
    <div v-if="create" class="flex justify-end pt-5">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="$emit('cancel')"
      >
        Cancel
      </button>
      <button
        type="submit"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
        @click.prevent="createEnvironment"
      >
        Create
      </button>
    </div>
    <!-- Update button group -->
    <div v-else class="flex justify-between items-center pt-5">
      <template v-if="state.environment.rowStatus == 'NORMAL'">
        <BBButtonConfirm
          v-if="allowArchive"
          :style="'ARCHIVE'"
          :buttonText="'Archive this environment'"
          :okText="'Archive'"
          :confirmTitle="`Archive environment '${state.environment.name}'?`"
          :confirmDescription="'Archived environment will not be shown on the normal interface. You can still restore later from the Archive page.'"
          :requireConfirm="true"
          @confirm="archiveEnvironment"
        />
      </template>
      <template v-else-if="state.environment.rowStatus == 'ARCHIVED'">
        <BBButtonConfirm
          v-if="allowRestore"
          :style="'RESTORE'"
          :buttonText="'Restore this environment'"
          :okText="'Restore'"
          :confirmTitle="`Restore environment '${state.environment.name}' to normal state?`"
          :confirmDescription="''"
          :requireConfirm="true"
          @confirm="restoreEnvironment"
        />
      </template>
      <div v-else></div>
      <div v-if="allowEdit">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          :disabled="!valueChanged"
          @click.prevent="revertEnvironment"
        >
          Revert
        </button>
        <button
          type="submit"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          :disabled="!valueChanged"
          @click.prevent="updateEnvironment"
        >
          Update
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive, PropType, watch } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import isEmpty from "lodash-es/isEmpty";
import { Environment, EnvironmentCreate, EnvironmentPatch } from "../types";
import { isDBAOrOwner } from "../utils";

interface LocalState {
  environment: Environment | EnvironmentCreate;
}

export default {
  name: "EnvironmentForm",
  emits: ["create", "update", "cancel", "archive", "restore"],
  props: {
    create: {
      type: Boolean,
      default: false,
    },
    environment: {
      required: true,
      type: Object as PropType<Environment | EnvironmentCreate>,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      environment: cloneDeep(props.environment),
    });

    watch(
      () => props.environment,
      (cur: Environment | EnvironmentCreate) => {
        state.environment = cloneDeep(cur);
      }
    );

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]();
    });

    const allowArchive = computed(() => {
      return allowEdit.value && environmentList.value.length > 1;
    });

    const allowRestore = computed(() => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowEdit = computed(() => {
      return (
        props.create ||
        ((state.environment as Environment).rowStatus == "NORMAL" &&
          isDBAOrOwner(currentUser.value.role))
      );
    });

    const valueChanged = computed(() => {
      return !isEqual(props.environment, state.environment);
    });

    const allowCreate = computed(() => {
      return !isEmpty(state.environment?.name);
    });

    const revertEnvironment = () => {
      state.environment = cloneDeep(props.environment!);
    };

    const createEnvironment = () => {
      emit("create", state.environment);
    };

    const updateEnvironment = () => {
      const patchedEnvironment: EnvironmentPatch = {};

      if (state.environment.name != props.environment!.name) {
        patchedEnvironment.name = state.environment.name;
      }
      if (
        state.environment.approvalPolicy != props.environment!.approvalPolicy
      ) {
        patchedEnvironment.approvalPolicy = state.environment.approvalPolicy;
      }
      emit("update", patchedEnvironment);
    };

    const archiveEnvironment = () => {
      emit("archive", state.environment);
    };

    const restoreEnvironment = () => {
      emit("restore", state.environment);
    };

    return {
      state,
      allowArchive,
      allowRestore,
      allowEdit,
      valueChanged,
      allowCreate,
      revertEnvironment,
      createEnvironment,
      updateEnvironment,
      archiveEnvironment,
      restoreEnvironment,
    };
  },
};
</script>
