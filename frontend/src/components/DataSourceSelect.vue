<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        $emit('select-data-source-id', parseInt(e.target.value));
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      <template v-if="!database">
        {{ $t("datasource.select-database-first") }}
      </template>
      <template v-else> {{ $t("datasource.select-data-source") }} </template>
    </option>
    <template v-if="database">
      <option
        v-for="(dataSource, index) in database.dataSourceList"
        :key="index"
        :value="dataSource.id"
        :selected="dataSource.id == state.selectedId"
      >
        <template v-if="dataSource.type == 'RO'">
          {{ dataSource.name }} ({{ $t("common.readonly") }})
        </template>
        <template v-else>
          {{ dataSource.name }}
        </template>
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import { defineComponent, PropType, reactive } from "vue";
import { Database } from "../types";

interface LocalState {
  selectedId?: number;
}

export default defineComponent({
  name: "DataSourceSelect",
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
  emits: ["select-data-source-id"],
  setup(props) {
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    return {
      state,
    };
  },
});
</script>
