<template>
  <form
    class="px-4 space-y-6 divide-y divide-gray-200"
    @submit.prevent="$emit('submit', state.environment)"
  >
    <div class="pt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
      <div class="sm:col-span-2">
        <label for="name" class="text-lg leading-6 font-medium text-control">
          Instance Name <span class="text-red-600">*</span>
        </label>
        <input
          required
          id="name"
          name="name"
          type="text"
          class="textfield mt-4 w-full"
          :value="state.dataSource.name"
          @input="updateDataSource('name', $event.target.value)"
        />
      </div>
    </div>
    <!-- Create button group -->
    <div class="flex justify-end pt-5">
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
  </form>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import { DataSourceNew } from "../types";

interface LocalState {
  dataSource: DataSourceNew;
}

export default {
  name: "DataSourceForm",
  emits: ["submit", "cancel", "delete"],
  props: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({
      dataSource: {
        name: "New Data Source",
        type: "RW",
      },
    });

    const updateDataSource = (field: string, value: string) => {
      if (state.dataSource) {
        (state.dataSource as any)[field] = value;
      }
    };

    return {
      state,
      updateDataSource,
    };
  },
};
</script>
