<template>
  <div>
    <BBTab
      :tabTitleList="tabTitleList"
      :selectedIndex="state.selectedIndex"
      :reorderModel="state.reorder ? 'ALWAYS' : 'NEVER'"
      @reorder-index="reorderEnvironment"
      @select-index="selectEnvironment"
    >
      <BBTabPanel
        v-for="(item, index) in environmentList"
        :key="item.id"
        :id="item.id"
        :active="index == state.selectedIndex"
      >
        <div v-if="state.reorder" class="flex justify-center pt-5">
          <button
            type="button"
            class="btn-normal py-2 px-4"
            @click.prevent="discardReorder"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
            :disabled="!orderChanged"
            @click.prevent="doReorder"
          >
            Apply new order
          </button>
        </div>
        <EnvironmentDetail
          v-else
          :allowDelete="environmentList.length > 1"
          :environment="item"
          @delete="doDelete"
        />
      </BBTabPanel>
    </BBTab>
  </div>
  <BBModal
    v-if="state.showCreateModal"
    :title="'Create Environment'"
    @close="state.showCreateModal = false"
  >
    <EnvironmentForm
      :create="true"
      @submit="doCreate"
      @cancel="state.showCreateModal = false"
    />
  </BBModal>
</template>

<script lang="ts">
import { onMounted, onUnmounted, watchEffect, computed, reactive } from "vue";
import { useStore } from "vuex";
import { array_swap } from "../utils";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import EnvironmentDetail from "../views/EnvironmentDetail.vue";
import { Environment, NewEnvironment } from "../types";

interface LocalState {
  reorderedEnvironmentList: Environment[];
  selectedIndex: number;
  showCreateModal: boolean;
  reorder: boolean;
}

export default {
  name: "EnvironmentDashboard",
  components: {
    EnvironmentDetail,
    EnvironmentForm,
  },
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      reorderedEnvironmentList: [],
      selectedIndex: -1,
      showCreateModal: false,
      reorder: false,
    });

    onMounted(() => {
      store.dispatch("command/registerCommand", {
        id: "bytebase.environment.add",
        registerId: "environment.dashboard",
        run: () => {
          createEnvironment();
        },
      });
      store.dispatch("command/registerCommand", {
        id: "bytebase.environment.reorder",
        registerId: "environment.dashboard",
        run: () => {
          startReorder();
        },
      });
    });

    onUnmounted(() => {
      store.dispatch("command/unregisterCommand", {
        id: "bytebase.environment.add",
        registerId: "environment.dashboard",
      });
      store.dispatch("command/unregisterCommand", {
        id: "bytebase.environment.reorder",
        registerId: "environment.dashboard",
      });
    });

    const prepareEnvironmentList = () => {
      store
        .dispatch("environment/fetchEnvironmentList")
        .then((list) => {
          if (list.length > 0) {
            selectEnvironment(0);
          }
        })
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareEnvironmentList);

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]();
    });

    const tabTitleList = computed(() => {
      if (environmentList.value) {
        if (state.reorder) {
          return state.reorderedEnvironmentList.map(
            (item: Environment) => item.attributes.name
          );
        }
        return environmentList.value.map(
          (item: Environment) => item.attributes.name
        );
      }
      return [];
    });

    const createEnvironment = () => {
      stopReorder();
      state.showCreateModal = true;
    };

    const doCreate = (newEnvironment: NewEnvironment) => {
      store
        .dispatch("environment/createEnvironment", {
          newEnvironment,
        })
        .then((createdEnvironment) => {
          state.showCreateModal = false;
          selectEnvironment(environmentList.value.length - 1);
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const startReorder = () => {
      state.reorderedEnvironmentList = [...environmentList.value];
      state.reorder = true;
    };

    const stopReorder = () => {
      state.reorder = false;
      state.reorderedEnvironmentList = [];
    };

    const reorderEnvironment = (sourceIndex: number, targetIndex: number) => {
      array_swap(state.reorderedEnvironmentList, sourceIndex, targetIndex);
    };

    const orderChanged = computed(() => {
      for (let i = 0; i < state.reorderedEnvironmentList.length; i++) {
        if (
          state.reorderedEnvironmentList[i].id != environmentList.value[i].id
        ) {
          return true;
        }
      }
      return false;
    });

    const discardReorder = () => {
      stopReorder();
    };

    const doReorder = () => {
      store
        .dispatch(
          "environment/reorderEnvironmentList",
          state.reorderedEnvironmentList
        )
        .then(() => {
          stopReorder();
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doDelete = (environment: Environment) => {
      store
        .dispatch("environment/deleteEnvironmentById", {
          id: environment.id,
        })
        .then(() => {
          if (environmentList.value.length > 0) {
            selectEnvironment(0);
          }
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const selectEnvironment = (index: number) => {
      state.selectedIndex = index;
    };

    const tabClass = computed(() => "w-1/" + environmentList.value.length);

    return {
      state,
      environmentList,
      tabTitleList,
      createEnvironment,
      doCreate,
      reorderEnvironment,
      orderChanged,
      discardReorder,
      doReorder,
      doDelete,
      selectEnvironment,
      tabClass,
    };
  },
};
</script>
