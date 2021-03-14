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
    <InstanceTable :instanceList="filteredList(state.instanceList)" />
  </div>
</template>

<script lang="ts">
import { watchEffect, onMounted, reactive, ref } from "vue";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import InstanceTable from "../components/InstanceTable.vue";
import { useStore } from "vuex";
import { Environment, Instance } from "../types";

interface LocalState {
  instanceList: Instance[];
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
      instanceList: [],
      searchText: "",
    });
    const store = useStore();

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

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

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

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

    watchEffect(prepareInstanceList);

    return {
      searchField,
      state,
      filteredList,
      selectEnvironment,
      changeSearchText,
    };
  },
};
</script>
