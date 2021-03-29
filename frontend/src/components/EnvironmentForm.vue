<template>
  <form
    class="px-4 space-y-6 divide-y divide-gray-200"
    @submit.prevent="$emit('submit', state.environment)"
  >
    <div class="pt-6 grid grid-cols-1 gap-y-6 gap-x-4">
      <div class="col-span-1">
        <label for="name" class="text-lg leading-6 font-medium text-control">
          Environment Name <span class="text-red-600">*</span>
        </label>
        <BBTextField
          class="mt-4 w-full"
          :disabled="!allowEdit"
          :required="true"
          :value="state.environment.name"
          @input="state.environment.name = $event.target.value"
        />
      </div>
    </div>
    <template v-if="allowEdit">
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
        >
          Create
        </button>
      </div>
      <!-- Update button group -->
      <div v-else class="flex justify-between items-center pt-5">
        <div>
          <BBButtonTrash
            v-if="allowDelete"
            :buttonText="'Delete this environment'"
            :requireConfirm="false"
            @confirm="$emit('delete')"
          />
        </div>
        <div>
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
          >
            Update
          </button>
        </div>
      </div>
    </template>
  </form>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import isEmpty from "lodash-es/isEmpty";
import { Environment, EnvironmentNew } from "../types";

interface LocalState {
  environment?: Environment | EnvironmentNew;
}

export default {
  name: "EnvironmentForm",
  emits: ["submit", "cancel", "delete"],
  props: {
    create: {
      type: Boolean,
      default: false,
    },
    allowDelete: {
      type: Boolean,
      default: true,
    },
    environment: {
      // Can be false when create is true
      required: false,
      type: Object as PropType<Environment>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const state = reactive<LocalState>({});

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    // [NOTE] Ternary operator doesn't trigger VS type checking, so we use a separate
    // IF block.
    if (props.environment) {
      state.environment = cloneDeep(props.environment);
    } else {
      state.environment = {
        name: "New Env",
        order: -1,
      };
    }

    const allowEdit = computed(() => {
      return (
        currentUser.value.role == "DBA" || currentUser.value.role == "OWNER"
      );
    });

    const valueChanged = computed(() => {
      return !isEqual(props.environment, state.environment);
    });

    const allowCreate = computed(() => {
      return !isEmpty(state.environment?.name);
    });

    const revertEnvironment = () => {
      state.environment = cloneDeep(props.environment);
    };

    return {
      state,
      allowEdit,
      valueChanged,
      allowCreate,
      revertEnvironment,
    };
  },
};
</script>
