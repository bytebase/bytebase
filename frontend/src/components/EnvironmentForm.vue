<template>
  <form
    class="px-4 space-y-6 divide-y divide-gray-200"
    @submit.prevent="$emit('submit', state.environment)"
  >
    <div class="pt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
      <div class="sm:col-span-2">
        <label for="name" class="text-lg leading-6 font-medium text-control">
          Environment Name <span class="text-red-600">*</span>
        </label>
        <input
          required
          id="name"
          name="name"
          type="text"
          class="textfield mt-4 w-full"
          :value="state.environment.name"
          @input="updateEnvironment('name', $event.target.value)"
        />
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
  </form>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
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
    const state = reactive<LocalState>({});

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

    const valueChanged = computed(() => {
      return !isEqual(props.environment, state.environment);
    });

    const updateEnvironment = (field: string, value: string) => {
      if (state.environment) {
        (state.environment as any)[field] = value;
      }
    };

    const revertEnvironment = () => {
      state.environment = cloneDeep(props.environment);
    };

    return {
      state,
      valueChanged,
      updateEnvironment,
      revertEnvironment,
    };
  },
};
</script>
