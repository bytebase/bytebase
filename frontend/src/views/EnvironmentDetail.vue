<template>
  <div class="py-2">
    <ArchiveBanner v-if="state.environment.rowStatus == 'ARCHIVED'" />
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
import ArchiveBanner from "../components/ArchiveBanner.vue";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import { Environment, EnvironmentPatch } from "../types";
import { idFromSlug } from "../utils";

export default {
  name: "EnvironmentDetail",
  emits: ["archive"],
  components: {
    ArchiveBanner,
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
