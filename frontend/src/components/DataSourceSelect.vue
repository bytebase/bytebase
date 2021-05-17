<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        $emit('select-data-source-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      <template v-if="!database"> Select database first </template>
      <template v-else> Select data source </template>
    </option>
    <template v-if="database">
      <option
        v-for="(dataSource, index) in database.dataSourceList"
        :key="index"
        :value="dataSource.id"
        :selected="dataSource.id == state.selectedId"
      >
        <template v-if="dataSource.type == 'RO'">
          {{ dataSource.name }} (readonly)
        </template>
        <template v-else>
          {{ dataSource.name }}
        </template>
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import { computed, PropType, reactive } from "vue";
import { useStore } from "vuex";
import { Database } from "../types";

interface LocalState {
  selectedId?: number;
}

export default {
  name: "DataSourceSelect",
  emits: ["select-data-source-id"],
  components: {},
  props: {
    selectedId: {
      type: Number,
    },
    database: {
      type: Object as PropType<Database>,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    return {
      state,
    };
  },
};
</script>
