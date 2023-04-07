<template>
  <div class="space-y-4 pb-4">
    <div>
      <EnvironmentTabFilter
        :environment="state.filter.environment?.id ?? UNKNOWN_ID"
        :include-all="true"
        @update:environment="changeEnvironment"
      />
    </div>
    <div class="w-[48rem] max-w-full">
      <SlowQueryPolicyTable
        :instance-list="state.ready ? filteredInstanceList : []"
        :policy-list="policyList"
        :toggle-active="toggleActive"
        :show-placeholder="state.ready"
      />
      <div
        v-if="!state.ready"
        class="relative flex flex-col h-[8rem] items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";

import {
  featureToRef,
  useEnvironmentList,
  useInstanceStore,
  useSlowQueryPolicyStore,
} from "@/store";
import {
  Environment,
  EnvironmentId,
  Instance,
  SlowQueryPolicyPayload,
  UNKNOWN_ID,
} from "@/types";
import { EnvironmentTabFilter } from "@/components/v2";
import { SlowQueryPolicyTable } from "./components";
import { instanceSupportSlowQuery } from "@/utils";

const emit = defineEmits<{
  (event: "show-feature-modal"): void;
}>();

type LocalState = {
  ready: boolean;
  instanceList: Instance[];
  filter: {
    environment: Environment | undefined;
  };
};

const state = reactive<LocalState>({
  ready: false,
  instanceList: [],
  filter: {
    environment: undefined,
  },
});

const policyStore = useSlowQueryPolicyStore();
const instanceStore = useInstanceStore();
const environmentList = useEnvironmentList(["NORMAL"]);
const hasSlowQueryFeature = featureToRef("bb.feature.slow-query");

const policyList = computed(() => {
  return policyStore.getPolicyListByResourceTypeAndPolicyType(
    "instance",
    "bb.policy.slow-query"
  );
});

const filteredInstanceList = computed(() => {
  const list = state.instanceList;
  const { environment } = state.filter;
  if (environment && environment.id !== UNKNOWN_ID) {
    return list.filter(
      (instance) => instance.environment.id === environment.id
    );
  }
  return list;
});

const prepare = async () => {
  try {
    const prepareInstanceList = async () => {
      const list = await instanceStore.fetchInstanceList(["NORMAL"]);
      state.instanceList = list.filter(instanceSupportSlowQuery);
    };
    const preparePolicyList = async () => {
      await policyStore.fetchPolicyListByResourceTypeAndPolicyType(
        "instance",
        "bb.policy.slow-query"
      );
    };
    await Promise.all([prepareInstanceList(), preparePolicyList()]);
  } finally {
    state.ready = true;
  }
};

const changeEnvironment = (id: EnvironmentId | undefined) => {
  state.filter.environment = environmentList.value.find((env) => env.id === id);
};

const toggleActive = async (instance: Instance, active: boolean) => {
  if (!hasSlowQueryFeature.value) {
    emit("show-feature-modal");
    return;
  }

  try {
    const payload: SlowQueryPolicyPayload = {
      active,
    };
    await policyStore.upsertPolicyByResourceTypeAndPolicyType(
      "instance",
      instance.id,
      "bb.policy.slow-query",
      {
        payload,
      }
    );
  } catch {
    // nothing
  }
};

onMounted(prepare);
</script>
