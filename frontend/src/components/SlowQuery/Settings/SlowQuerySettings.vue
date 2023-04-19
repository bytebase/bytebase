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
        :toggle-instance-active="toggleInstanceActive"
        :toggle-database-active="toggleDatabaseActive"
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
  useDatabaseStore,
  useEnvironmentList,
  useInstanceStore,
  useSlowQueryPolicyStore,
  useSlowQueryStore,
} from "@/store";
import {
  Database,
  Environment,
  EnvironmentId,
  Instance,
  SlowQueryPolicyPayload,
  UNKNOWN_ID,
  engineName,
} from "@/types";
import { EnvironmentTabFilter } from "@/components/v2";
import { SlowQueryPolicyTable } from "./components";
import {
  InstanceListSupportSlowQuery,
  extractSlowQueryPolicyPayload,
  instanceSupportSlowQuery,
  slowQueryTypeOfInstance,
} from "@/utils";
import { useI18n } from "vue-i18n";
import { cloneDeep } from "lodash-es";

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
const databaseStore = useDatabaseStore();
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
      const instanceListWithDatabases = list.filter(
        (instance) => slowQueryTypeOfInstance(instance) === "DATABASE"
      );
      await instanceListWithDatabases.map((instance) =>
        databaseStore.fetchDatabaseListByInstanceId(instance.id)
      );

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

const patchInstanceSlowQueryPolicy = (
  instance: Instance,
  active: boolean,
  databaseList: string[] = []
) => {
  const payload: SlowQueryPolicyPayload = {
    active,
    databaseList,
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

const toggleInstanceActive = async (instance: Instance, active: boolean) => {
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

const toggleDatabaseActive = async (
  instance: Instance,
  database: Database,
  active: boolean
) => {
  const composePolicyPayload = (
    instance: Instance,
    database: Database,
    active: boolean
  ) => {
    const payload = cloneDeep(
      extractSlowQueryPolicyPayload(
        policyList.value.find((policy) => policy.resourceId === instance.id)
      )
    );
    const databaseList = payload.databaseList ?? [];
    if (active) {
      if (!databaseList.includes(database.name)) {
        databaseList.push(database.name);
      }
    } else {
      const index = databaseList.indexOf(database.name);
      if (index >= 0) {
        databaseList.splice(index, 1);
      }
    }

    return {
      active: databaseList.length > 0,
      databaseList,
    };
  };
  try {
    const payload = composePolicyPayload(instance, database, active);
    await patchInstanceSlowQueryPolicy(
      instance,
      payload.active,
      payload.databaseList
    );
    if (active) {
      // When turning ON an database's slow query, call the corresponding
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

        const rollbackPayload = composePolicyPayload(instance, database, false);
        await patchInstanceSlowQueryPolicy(
          instance,
          rollbackPayload.active,
          rollbackPayload.databaseList
        );
      }
    }
  } catch {
    // nothing
  }
};

onMounted(prepare);

const attentionDescription = computed(() => {
  const versions = InstanceListSupportSlowQuery.map(([engine, minVersion]) => {
    const parts = [engineName(engine)];
    if (minVersion !== "0") {
      parts.push(minVersion);
    }
    return parts.join(" >= ");
  }).join(", ");

  return t("slow-query.attention-description", {
    versions,
  });
});
</script>
