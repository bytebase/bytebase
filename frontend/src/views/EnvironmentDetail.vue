<template>
  <div
    v-if="state.environment.rowStatus == 'ARCHIVED'"
    class="h-10 w-full text-2xl font-bold bg-gray-700 text-white flex justify-center items-center"
  >
    Archived
  </div>
  <EnvironmentForm
    v-if="state.environment"
    :environment="state.environment"
    @update="doUpdate"
    @archive="doArchive"
    @restore="doRestore"
  />
</template>

<script lang="ts">
import { reactive } from "vue";
import { useStore } from "vuex";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import { Environment, EnvironmentPatch } from "../types";
import { idFromSlug } from "../utils";

export default {
  name: "EnvironmentDetail",
  emits: ["archive"],
  components: {
    EnvironmentForm,
  },
  props: {
    environmentSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();

    const state = reactive({
      environment: store.getters["environment/environmentById"](
        idFromSlug(props.environmentSlug)
      ),
      showArchiveModal: false,
    });

    const assignEnvironment = (environment: Environment) => {
      state.environment = environment;
    };

    const doUpdate = (environmentPatch: EnvironmentPatch) => {
      store
        .dispatch("environment/patchEnvironment", {
          environmentId: idFromSlug(props.environmentSlug),
          environmentPatch,
        })
        .then((environment) => {
          assignEnvironment(environment);
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doArchive = (environment: Environment) => {
      store
        .dispatch("environment/patchEnvironment", {
          environmentId: environment.id,
          environmentPatch: {
            rowStatus: "ARCHIVED",
          },
        })
        .then((environment) => {
          emit("archive", environment);
          assignEnvironment(environment);
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doRestore = (environment: Environment) => {
      store
        .dispatch("environment/patchEnvironment", {
          environmentId: environment.id,
          environmentPatch: {
            rowStatus: "NORMAL",
          },
        })
        .then((environment) => {
          assignEnvironment(environment);
        })
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      state,
      doUpdate,
      doArchive,
      doRestore,
    };
  },
};
</script>
