<template>
  <EnvironmentForm
    v-if="state.environment"
    :environment="state.environment"
    @submit="doUpdate"
    @delete="state.showDeleteModal = true"
  />
  <BBAlert
    v-if="state.showDeleteModal"
    :style="'CRITICAL'"
    :okText="'Delete'"
    :title="
      'Delete environment \'' + state.environment.attributes.name + '\' ?'
    "
    @ok="
      () => {
        state.showDeleteModal = false;
        doDelete();
      }
    "
    @cancel="state.showDeleteModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { watchEffect, computed, inject, reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import { Environment, EnvironmentId } from "../types";
import environment from "../store/modules/environment";

export default {
  name: "EnvironmentDetail",
  emits: ["delete"],
  components: {
    EnvironmentForm,
  },
  props: {
    environment: {
      required: true,
      type: Object as PropType<Environment>,
    },
  },
  setup(props, { emit }) {
    const store = useStore();

    const state = reactive({
      environment: props.environment,
      showDeleteModal: false,
    });

    const router = useRouter();

    const assignEnvironment = (environment: Environment) => {
      state.environment = environment;
    };

    const doUpdate = (newEnvironment: Environment) => {
      store
        .dispatch("environment/patchEnvironmentById", {
          environmentId: props.environment.id,
          environment: newEnvironment,
        })
        .then((environment) => {
          assignEnvironment(environment);
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doDelete = () => {
      emit("delete", props.environment);
    };

    return {
      state,
      doUpdate,
      doDelete,
    };
  },
};
</script>
