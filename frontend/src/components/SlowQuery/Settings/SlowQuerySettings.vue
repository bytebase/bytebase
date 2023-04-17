<template>
  <div class="space-y-4 pb-4 w-[48rem] max-w-full">
    <div>
      <BBAttention :style="'WARN'" :description="attentionDescription" />
    </div>
    <div>
      <EnvironmentTabFilter
        :environment="state.filter.environment?.id ?? UNKNOWN_ID"
        :include-all="true"
        @update:environment="changeEnvironment"
      />
    </div>
    <div>
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

import { BBAttention } from "@/bbkit";
import {
  pushNotification,
  useEnvironmentList,
  useInstanceStore,
  useSlowQueryPolicyStore,
  useSlowQueryStore,
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
import { useI18n } from "vue-i18n";

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

const { t } = useI18n();
const policyStore = useSlowQueryPolicyStore();
const slowQueryStore = useSlowQueryStore();
const instanceStore = useInstanceStore();
const environmentList = useEnvironmentList(["NORMAL"]);

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

const patchInstanceSlowQueryPolicy = (instance: Instance, active: boolean) => {
  const payload: SlowQueryPolicyPayload = {
    active,
  };
  return policyStore.upsertPolicyByResourceTypeAndPolicyType(
    "instance",
    instance.id,
    "bb.policy.slow-query",
    {
      payload,
    }
  );
};

const toggleActive = async (instance: Instance, active: boolean) => {
  try {
    await patchInstanceSlowQueryPolicy(instance, active);
    if (active) {
      // When turning ON an instance's slow query, call the corresponding
      // API endpoint to sync slow queries from the instance immediately.
      try {
        await slowQueryStore.syncSlowQueriesByInstance(instance);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } catch (err: any) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: typeof err.message === "string" ? err.message : String(err),
        });

        await patchInstanceSlowQueryPolicy(instance, false);
      }
    }
  } catch {
    // nothing
  }
};

onMounted(prepare);

const attentionDescription = computed(() => {
  const versions = `MySQL >= 5.7`;

  return t("slow-query.attention-description", {
    versions,
  });
});
</script>
