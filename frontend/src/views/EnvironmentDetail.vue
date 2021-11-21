<template>
  <div class="py-2">
    <ArchiveBanner v-if="state.environment.rowStatus == 'ARCHIVED'" />
  </div>
  <EnvironmentForm
    v-if="state.approvalPolicy && state.backupPolicy"
    :environment="state.environment"
    :approvalPolicy="state.approvalPolicy"
    :backupPolicy="state.backupPolicy"
    @update="doUpdate"
    @archive="doArchive"
    @restore="doRestore"
    @update-policy="updatePolicy"
  />
</template>

<script lang="ts">
import { reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import {
  Environment,
  EnvironmentID,
  EnvironmentPatch,
  Policy,
  PolicyType,
  unknown,
} from "../types";
import { idFromSlug } from "../utils";

interface LocalState {
  environment: Environment;
  showArchiveModal: boolean;
  approvalPolicy?: Policy;
  backupPolicy?: Policy;
}

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

    const state = reactive<LocalState>({
      environment: store.getters["environment/environmentByID"](
        idFromSlug(props.environmentSlug)
      ),
      showArchiveModal: false,
    });

    const preparePolicy = () => {
      store
        .dispatch("policy/fetchPolicyByEnvironmentAndType", {
          environmentID: (state.environment as Environment).id,
          type: "bb.policy.pipeline-approval",
        })
        .then((policy: Policy) => {
          state.approvalPolicy = policy;
        });

      store
        .dispatch("policy/fetchPolicyByEnvironmentAndType", {
          environmentID: (state.environment as Environment).id,
          type: "bb.policy.backup-plan",
        })
        .then((policy: Policy) => {
          state.backupPolicy = policy;
        });
    };

    watchEffect(preparePolicy);

    const assignEnvironment = (environment: Environment) => {
      state.environment = environment;
    };

    const doUpdate = (environmentPatch: EnvironmentPatch) => {
      store
        .dispatch("environment/patchEnvironment", {
          environmentID: idFromSlug(props.environmentSlug),
          environmentPatch,
        })
        .then((environment) => {
          assignEnvironment(environment);
        });
    };

    const doArchive = (environment: Environment) => {
      store
        .dispatch("environment/patchEnvironment", {
          environmentID: environment.id,
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
          environmentID: environment.id,
          environmentPatch: {
            rowStatus: "NORMAL",
          },
        })
        .then((environment) => {
          assignEnvironment(environment);
        });
    };

    const updatePolicy = (
      environmentID: EnvironmentID,
      type: PolicyType,
      policy: Policy
    ) => {
      store
        .dispatch("policy/upsertPolicyByEnvironmentAndType", {
          environmentID,
          type: type,
          policyUpsert: {
            payload: policy.payload,
          },
        })
        .then((policy: Policy) => {
          state.backupPolicy = policy;
        });
    };

    return {
      state,
      doUpdate,
      doArchive,
      doRestore,
      updatePolicy,
    };
  },
};
</script>
