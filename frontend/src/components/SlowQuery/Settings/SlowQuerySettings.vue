<template>
  <div class="space-y-4 pb-4 max-w-full">
    <div class="textinfolabel">
      {{ $t("slow-query.attention-description") }}
      <a
        href="https://www.bytebase.com/docs/slow-query/overview?source=console"
        target="_blank"
        class="normal-link inline-flex flex-row items-center"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>
    <div class="flex items-center justify-between">
      <EnvironmentTabFilter
        :environment="state.filter.environment?.name"
        :include-all="true"
        @update:environment="changeEnvironment"
      />
      <SearchBox v-model:value="state.filter.keyword" />
    </div>
    <div>
      <SlowQueryPolicyTable
        :composed-slow-query-policy-list="filteredComposedSlowQueryPolicyList"
        :policy-list="policyList"
        :toggle-active="toggleActive"
        :ready="state.ready"
        :show-placeholder="true"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { EnvironmentTabFilter, SearchBox } from "@/components/v2";
import {
  pushNotification,
  useEnvironmentV1List,
  useInstanceV1Store,
  useSlowQueryPolicyStore,
  useSlowQueryStore,
} from "@/store";
import { useGracefulRequest } from "@/store/modules/utils";
import { ComposedInstance, ComposedSlowQueryPolicy, UNKNOWN_ID } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { instanceV1SupportSlowQuery } from "@/utils";
import { SlowQueryPolicyTable } from "./components";

type LocalState = {
  ready: boolean;
  instanceList: ComposedInstance[];
  filter: {
    environment: Environment | undefined;
    keyword: string;
  };
};

const state = reactive<LocalState>({
  ready: false,
  instanceList: [],
  filter: {
    environment: undefined,
    keyword: "",
  },
});

const { t } = useI18n();
const slowQueryPolicyStore = useSlowQueryPolicyStore();
const slowQueryStore = useSlowQueryStore();
const instanceV1Store = useInstanceV1Store();
const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const policyList = computed(() => {
  return slowQueryPolicyStore.getPolicyList();
});

const composedSlowQueryPolicyList = computed(() => {
  const list = state.instanceList.map<ComposedSlowQueryPolicy>((instance) => {
    const policy = policyList.value.find((p) => p.resourceUid == instance.uid);
    return {
      instance,
      active: policy?.slowQueryPolicy?.active ?? false,
    };
  });

  return orderBy(
    list,
    [
      (item) => item.instance.engine,
      (item) => item.instance.environmentEntity.order,
      (item) => item.instance.name,
    ],
    ["asc", "desc", "asc"]
  );
});

const filteredComposedSlowQueryPolicyList = computed(() => {
  let list = [...composedSlowQueryPolicyList.value];
  const { environment } = state.filter;
  if (environment && environment.uid !== String(UNKNOWN_ID)) {
    list = list.filter(
      (item) => String(item.instance.environment) === environment.name
    );
  }
  const keyword = state.filter.keyword.trim().toLowerCase();
  if (keyword) {
    list = list.filter((item) =>
      item.instance.name.toLowerCase().includes(keyword)
    );
  }

  return list;
});

const prepare = async () => {
  try {
    const prepareInstanceList = async () => {
      const list = await instanceV1Store.fetchInstanceList(
        false /* !showDeleted */
      );
      state.instanceList = list.filter(instanceV1SupportSlowQuery);
    };
    const preparePolicyList = async () => {
      await slowQueryPolicyStore.fetchPolicyList();
    };
    await Promise.all([prepareInstanceList(), preparePolicyList()]);
  } finally {
    state.ready = true;
  }
};

const changeEnvironment = (name: string | undefined) => {
  state.filter.environment = environmentList.value.find(
    (env) => env.name === name
  );
};

const patchInstanceSlowQueryPolicy = async (
  instance: ComposedInstance,
  active: boolean
) => {
  return slowQueryPolicyStore.upsertPolicy({
    parentPath: instance.name,
    active,
  });
};

const toggleActive = async (instance: ComposedInstance, active: boolean) => {
  try {
    await patchInstanceSlowQueryPolicy(instance, active);
    if (active) {
      // When turning ON an instance's slow query, call the corresponding
      // API endpoint to sync slow queries from the instance immediately.
      try {
        await useGracefulRequest(() =>
          slowQueryStore.syncSlowQueries(instance.name)
        );
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } catch {
        await patchInstanceSlowQueryPolicy(instance, false);
      }
    }
  } catch {
    // nothing
  }
};

onMounted(prepare);
</script>
