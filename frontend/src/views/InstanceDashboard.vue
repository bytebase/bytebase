<template>
  <div class="flex flex-col">
    <div class="px-5 py-2 flex justify-between items-center">
      <!-- eslint-disable vue/attribute-hyphenation -->
      <EnvironmentTabFilter
        :selectedId="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('instance.search-instance-name')"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <InstanceTable :instance-list="filteredList(instanceList)" />
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, ref, defineComponent } from "vue";
import { useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import InstanceTable from "../components/InstanceTable.vue";
import { Environment, Instance } from "../types";
import { cloneDeep } from "lodash-es";
import { sortInstanceList } from "../utils";
import {
  useUIStateStore,
  useEnvironmentStore,
  useEnvironmentList,
  useInstanceList,
} from "@/store";

interface LocalState {
  searchText: string;
  selectedEnvironment?: Environment;
}

export default defineComponent({
  name: "InstanceDashboard",
  components: {
    EnvironmentTabFilter,
    InstanceTable,
  },
  setup() {
    const searchField = ref();

    const uiStateStore = useUIStateStore();
    const router = useRouter();

    const environmentList = useEnvironmentList(["NORMAL"]);

    const state = reactive<LocalState>({
      searchText: "",
      selectedEnvironment: router.currentRoute.value.query.environment
        ? useEnvironmentStore().getEnvironmentById(
            parseInt(router.currentRoute.value.query.environment as string, 10)
          )
        : undefined,
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!uiStateStore.getIntroStateByKey("instance.visit")) {
        uiStateStore.saveIntroStateByKey({
          key: "instance.visit",
          newState: true,
        });
      }
    });

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

    const rawInstanceList = useInstanceList();

    const instanceList = computed(() => {
      return sortInstanceList(
        cloneDeep(rawInstanceList.value),
        environmentList.value
      );
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
});
</script>
