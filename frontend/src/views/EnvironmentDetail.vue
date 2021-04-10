<template>
  <EnvironmentForm
    v-if="state.environment"
    :allowDelete="allowDelete"
    :environment="state.environment"
    @update="doUpdate"
    @delete="state.showDeleteModal = true"
  />
  <BBAlert
    v-if="state.showDeleteModal"
    :style="'CRITICAL'"
    :okText="'Delete'"
    :title="'Delete environment \'' + state.environment.name + '\' ?'"
    :description="'You cannot undo this action.'"
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
import { reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import { Environment, EnvironmentPatch } from "../types";

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
    allowDelete: {
      type: Boolean,
      default: true,
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

    const doUpdate = (environmentPatch: EnvironmentPatch) => {
      store
        .dispatch("environment/patchEnvironment", {
          environmentId: props.environment.id,
          environmentPatch,
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
