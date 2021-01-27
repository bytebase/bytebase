<template>
  <div>
    <BBTab
      :itemList="tabItemList"
      :selectedId="state.selectedId"
      allowReorder
      @create-item="createEnvironment"
      @reorder-item="redorderEnvironment"
      @select-item="selectEnvironment"
    >
      <BBTabPanel
        v-for="item in environmentList"
        :key="item.id"
        :id="item.id"
        :active="item.id == state.selectedId"
      >
        <EnvironmentDetail :environment="item" @delete="deleteEnvironment" />
      </BBTabPanel>
    </BBTab>
  </div>
  <BBModal
    :showing="state.showCreateModal"
    :title="'Create Environment'"
    @close="state.showCreateModal = false"
  >
    <EnvironmentForm
      :create="true"
      @submit="tryCreate"
      @cancel="state.showCreateModal = false"
    />
  </BBModal>
</template>

<script lang="ts">
import { watchEffect, computed, inject, reactive, readonly } from "vue";
import { useStore } from "vuex";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import EnvironmentDetail from "../views/EnvironmentDetail.vue";
import { Environment, NewEnvironment } from "../types";

interface LocalState {
  selectedId: string;
  showCreateModal: boolean;
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
      selectedId: "",
      showCreateModal: false,
    });

    const prepareEnvironmentList = () => {
      store
        .dispatch("environment/fetchEnvironmentList")
        .then((list) => {
          if (list.length > 0) {
            selectEnvironment(list[0].id);
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

    const tabItemList = computed(() => {
      if (environmentList.value) {
        return environmentList.value?.map((item: Environment) => ({
          id: item.id,
          name: item.attributes.name,
        }));
      }
      return new Array();
    });

    const tryCreate = (newEnvironment: NewEnvironment) => {
      store
        .dispatch("environment/createEnvironment", {
          newEnvironment,
        })
        .then((createdEnvironment) => {
          state.showCreateModal = false;
          selectEnvironment(createdEnvironment.id);
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const createEnvironment = () => {
      state.showCreateModal = true;
    };

    const redorderEnvironment = (sourceIndex: number, targetIndex: number) => {
      store
        .dispatch("environment/reorderEnvironmentList", {
          sourceIndex,
          targetIndex,
        })
        .then((list) => {})
        .catch((error) => {
          console.log(error);
        });
    };

    const deleteEnvironment = (environment: Environment) => {
      store
        .dispatch("environment/deleteEnvironmentById", {
          id: environment.id,
        })
        .then(() => {
          if (tabItemList.value.length > 0) {
            selectEnvironment(tabItemList.value[0].id);
          }
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const selectEnvironment = (id: string) => {
      state.selectedId = id;
    };

    const tabClass = computed(() => "w-1/" + environmentList.value.length);

    return {
      state,
      environmentList,
      tabItemList,
      tryCreate,
      createEnvironment,
      redorderEnvironment,
      deleteEnvironment,
      selectEnvironment,
      tabClass,
    };
  },
};
</script>
