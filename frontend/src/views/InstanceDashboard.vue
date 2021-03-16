<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-between items-center">
      <EnvironmentTabFilter @select-environment="selectEnvironment" />
      <BBTableSearch
        ref="searchField"
        :placeholder="'Search instance name'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <InstanceTable :instanceList="filteredList(instanceList)" />
  </div>
</template>

<script lang="ts">
import { computed, watchEffect, onMounted, reactive, ref } from "vue";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import InstanceTable from "../components/InstanceTable.vue";
import { useStore } from "vuex";
import { Environment, Instance } from "../types";

interface LocalState {
  selectedEnvironment?: Environment;
  searchText: string;
}

export default {
  name: "InstanceDashboard",
  components: {
    EnvironmentTabFilter,
    InstanceTable,
  },
  setup(props, ctx) {
    const searchField = ref();

    const state = reactive<LocalState>({
      searchText: "",
    });
    const store = useStore();

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    const prepareInstanceList = () => {
      store.dispatch("instance/fetchInstanceList").catch((error) => {
        console.error(error);
      });
    };

    watchEffect(prepareInstanceList);

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const instanceList = computed(() => {
      return store.getters["instance/instanceList"]();
    });

    const filteredList = (list: Instance[]) => {
      if (!state.selectedEnvironment && !state.searchText) {
        // Select "All"
        return list;
      }
      return list.filter((instance) => {
        return (
          (!state.selectedEnvironment ||
            instance.environment.id == state.selectedEnvironment.id) &&
          (!state.searchText ||
            instance.name
              .toLowerCase()
              .includes(state.searchText.toLowerCase()))
        );
      });
    };

    return {
      searchField,
      state,
      instanceList,
      filteredList,
      selectEnvironment,
      changeSearchText,
    };
  },
};
</script>
