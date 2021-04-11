<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :selectedId="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="'Search instance name'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <InstanceTable :instanceList="filteredList(instanceList)" />
  </div>

  <BBAlert
    v-if="state.showGuide"
    :style="'INFO'"
    :okText="'Do not show again'"
    :cancelText="'Dismiss'"
    :title="'How to setup \'Instance\' ?'"
    :description="'Each instance in Bytebase usually maps to one of your database instance represented by an host:port address. This could be your on-premise MySQL instance, a RDS instance.\n\nBytebase requires read/write (NOT the super privilege) access to the instance in order to perform database operations on behalf of the user.\n\nInstance is only exposed to Owners and DBAs. For the developers, they are interacting directly with the database inside the instance.'"
    @ok="
      () => {
        doDismissGuide();
      }
    "
    @cancel="state.showGuide = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { computed, watchEffect, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import InstanceTable from "../components/InstanceTable.vue";
import { useStore } from "vuex";
import { Environment, Instance } from "../types";

interface LocalState {
  searchText: string;
  selectedEnvironment?: Environment;
  showGuide: boolean;
}

export default {
  name: "InstanceDashboard",
  components: {
    EnvironmentTabFilter,
    InstanceTable,
  },
  setup(props, ctx) {
    const searchField = ref();

    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      searchText: "",
      selectedEnvironment: router.currentRoute.value.query.environment
        ? store.getters["environment/environmentById"](
            router.currentRoute.value.query.environment
          )
        : undefined,
      showGuide: false,
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!store.getters["uistate/introStateByKey"]("guide.instance")) {
        setTimeout(() => {
          state.showGuide = true;
        }, 1000);
      }
    });

    const prepareInstanceList = () => {
      store.dispatch("instance/fetchInstanceList").catch((error) => {
        console.error(error);
      });
    };

    watchEffect(prepareInstanceList);

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
      if (environment) {
        router.replace({
          name: "workspace.instance",
          query: { environment: environment.id },
        });
      } else {
        router.replace({ name: "workspace.instance" });
      }
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const doDismissGuide = () => {
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "guide.instance",
        newState: true,
      });
      state.showGuide = false;
    };

    const instanceList = computed(() => {
      return store.getters["instance/instanceList"]("NORMAL");
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
      doDismissGuide,
    };
  },
};
</script>
