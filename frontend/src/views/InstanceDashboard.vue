<template>
  <div class="flex flex-col">
    <div class="px-2 py-1">
      <EnvironmentTabFilter @select-environment="selectEnvironment" />
    </div>
    <InstanceTable :instanceList="filteredList(state.instanceList)" />
  </div>
</template>

<script lang="ts">
import { watchEffect, reactive } from "vue";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import InstanceTable from "../components/InstanceTable.vue";
import { useStore } from "vuex";
import { Environment, Instance } from "../types";

interface LocalState {
  instanceList: Instance[];
  selectedEnvironment?: Environment;
}

export default {
  name: "InstanceDashboard",
  components: {
    EnvironmentTabFilter,
    InstanceTable,
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      instanceList: [],
    });
    const store = useStore();

    const prepareInstanceList = () => {
      store
        .dispatch("instance/fetchInstanceList")
        .then((instanceList: Instance[]) => {
          state.instanceList = instanceList;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
    };

    const filteredList = (list: Instance[]) => {
      if (!state.selectedEnvironment) {
        // Select "All"
        return list;
      }
      return list.filter((instance) => {
        return (
          instance.attributes.environmentId == state.selectedEnvironment!.id
        );
      });
    };

    watchEffect(prepareInstanceList);

    return {
      state,
      filteredList,
      selectEnvironment,
    };
  },
};
</script>
