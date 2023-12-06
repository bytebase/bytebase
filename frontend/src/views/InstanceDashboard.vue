<template>
  <div class="flex flex-col space-y-4">
    <FeatureAttention
      v-if="remainingInstanceCount <= 3"
      feature="bb.feature.instance-count"
      :description="instanceCountAttention"
    />
    <AdvancedSearchBox
      v-model:params="state.params"
      class="px-4"
      :autofocus="false"
      :placeholder="$t('instance.filter-instance-name')"
      :support-option-id-list="supportOptionIdList"
    />
    <InstanceV1Table
      :allow-selection="true"
      :instance-list="filteredInstanceV1List"
      :can-assign-license="subscriptionStore.currentPlan !== PlanType.FREE"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { InstanceV1Table } from "@/components/v2";
import {
  useUIStateStore,
  useSubscriptionV1Store,
  useEnvironmentV1List,
  useInstanceV1List,
  useInstanceV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";
import {
  sortInstanceV1ListByEnvironmentV1,
  extractEnvironmentResourceName,
  SearchParams,
  SearchScopeId,
} from "@/utils";

interface LocalState {
  params: SearchParams;
}

const subscriptionStore = useSubscriptionV1Store();
const instanceV1Store = useInstanceV1Store();
const uiStateStore = useUIStateStore();
const { t } = useI18n();

const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [
      {
        id: "environment",
        value: String(UNKNOWN_ID),
      },
    ],
  },
});

const selectedEnvironment = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "environment")?.value ??
    `${UNKNOWN_ID}`
  );
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("instance.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "instance.visit",
      newState: true,
    });
  }
});

const { instanceList: rawInstanceV1List } = useInstanceV1List(
  false /* !showDeleted */
);

const filteredInstanceV1List = computed(() => {
  let list = [...rawInstanceV1List.value];
  if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (instance) =>
        extractEnvironmentResourceName(instance.environment) ===
        selectedEnvironment.value
    );
  }
  const keyword = state.params.query.trim().toLowerCase();
  if (keyword) {
    list = list.filter((instance) =>
      instance.title.toLowerCase().includes(keyword)
    );
  }

  return sortInstanceV1ListByEnvironmentV1(list, environmentList.value);
});

const remainingInstanceCount = computed((): number => {
  return Math.max(
    0,
    subscriptionStore.instanceCountLimit -
      instanceV1Store.activeInstanceList.length
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

const supportOptionIdList = computed(
  (): { id: SearchScopeId; includeAll: boolean }[] => {
    return [
      {
        id: "environment",
        includeAll: true,
      },
    ];
  }
);
</script>
