<template>
  <div class="flex flex-col space-y-4">
    <FeatureAttention
      v-if="remainingInstanceCount <= 3"
      feature="bb.feature.instance-count"
      :description="instanceCountAttention"
    />
    <AdvancedSearch
      v-model:params="state.params"
      class="px-4"
      :autofocus="false"
      :placeholder="$t('instance.filter-instance-name')"
      :scope-options="scopeOptions"
    />
    <InstanceV1Table
      :bordered="false"
      :loading="!ready"
      :instance-list="filteredInstanceV1List"
      :can-assign-license="subscriptionStore.currentPlan !== PlanType.FREE"
      :default-expand-data-source="state.dataSourceToggle"
    />
  </div>
</template>

<script lang="tsx" setup>
import { NTag } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import AdvancedSearch from "@/components/AdvancedSearch";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { FeatureAttention } from "@/components/FeatureGuard";
import { InstanceV1Table } from "@/components/v2";
import {
  useUIStateStore,
  useSubscriptionV1Store,
  useEnvironmentV1List,
  useInstanceV1List,
  useInstanceV1Store,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import {
  type SearchParams,
  hostPortOfDataSource,
  readableDataSourceType,
  sortInstanceV1ListByEnvironmentV1,
  extractEnvironmentResourceName,
} from "@/utils";

interface LocalState {
  params: SearchParams;
  dataSourceToggle: string[];
}

const { t } = useI18n();
const subscriptionStore = useSubscriptionV1Store();
const instanceV1Store = useInstanceV1Store();
const uiStateStore = useUIStateStore();
const environmentList = useEnvironmentV1List();
const { instanceList, ready } = useInstanceV1List();

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  dataSourceToggle: [],
});

const selectedEnvironment = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "environment")?.value ?? ""
  );
});

const selectedAddress = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "address")?.value ?? ""
  );
});

watch(
  () => selectedAddress.value,
  (selectedAddress) => {
    if (!selectedAddress) {
      state.dataSourceToggle = [];
    }
  }
);

const addressOptions = computed(() => {
  const addressMap: Map<
    string,
    {
      keywords: string[];
      types: Set<string>;
    }
  > = new Map();

  for (const instance of instanceList.value) {
    for (const ds of instance.dataSources) {
      const host = hostPortOfDataSource(ds);
      if (!host) {
        continue;
      }
      if (!addressMap.has(host)) {
        addressMap.set(host, {
          keywords: [ds.host, ds.port],
          types: new Set(),
        });
      }
      addressMap.get(host)?.types?.add(readableDataSourceType(ds.type));
    }
  }

  const options: ValueOption[] = [];
  for (const [host, item] of addressMap.entries()) {
    options.push({
      value: host,
      keywords: [...item.keywords, ...item.types],
      render: () => {
        return (
          <div class={"flex items-center gap-x-2"}>
            {host}
            <div class={"flex items-center gap-x-1"}>
              {[...item.types].map((type) => (
                <NTag size="small" round>
                  {type}
                </NTag>
              ))}
            </div>
          </div>
        );
      },
    });
  }

  return options;
});

const scopeOptions = computed((): ScopeOption[] => {
  return [
    ...useCommonSearchScopeOptions(
      computed(() => state.params),
      ["environment"]
    ).value,
    {
      id: "address",
      title: t("instance.advanced-search.scope.address.title"),
      description: t("instance.advanced-search.scope.address.description"),
      options: addressOptions.value,
    },
  ];
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("instance.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "instance.visit",
      newState: true,
    });
  }
});

const filteredInstanceV1List = computed(() => {
  const keyword = state.params.query.trim().toLowerCase();
  const list = instanceList.value.filter((instance) => {
    if (keyword) {
      if (!instance.title.toLowerCase().includes(keyword)) {
        return false;
      }
    }
    if (selectedEnvironment.value) {
      if (
        extractEnvironmentResourceName(instance.environment) !==
        selectedEnvironment.value
      ) {
        return false;
      }
    }
    if (selectedAddress.value) {
      const matched = instance.dataSources.some(
        (ds) => hostPortOfDataSource(ds) === selectedAddress.value
      );
      if (matched) {
        state.dataSourceToggle.push(instance.name);
      }
      return matched;
    }
    return true;
  });

  return sortInstanceV1ListByEnvironmentV1(list, environmentList.value);
});

const remainingInstanceCount = computed((): number => {
  return Math.max(
    0,
    subscriptionStore.instanceCountLimit - instanceV1Store.instanceList.length
  );
});

const instanceCountAttention = computed((): string => {
  const upgrade = t("subscription.features.bb-feature-instance-count.upgrade");
  let status = "";

  if (remainingInstanceCount.value > 0) {
    status = t("subscription.features.bb-feature-instance-count.remaining", {
      total: subscriptionStore.instanceCountLimit,
      count: remainingInstanceCount.value,
    });
  } else {
    status = t("subscription.features.bb-feature-instance-count.runoutof", {
      total: subscriptionStore.instanceCountLimit,
    });
  }

  return `${status} ${upgrade}`;
});
</script>
