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
      Select data source
    </option>
    <option
      v-for="(dataSource, index) in dataSourceList"
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
  </select>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";

interface LocalState {
  selectedId?: string;
}

export default {
  name: "DataSourceSelect",
  emits: ["select-data-source-id"],
  components: {},
  props: {
    selectedId: {
      type: String,
    },
    instanceId: {
      required: true,
      type: String,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const dataSourceList = computed(() => {
      return store.getters["dataSource/dataSourceListByInstanceId"](
        props.instanceId
      );
    });

    return {
      state,
      dataSourceList,
    };
  },
};
</script>
