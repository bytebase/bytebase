<template>
  <div class="flex flex-col">
    <FeatureAttention
      v-if="remainingInstanceCount <= 3"
      custom-class="m-5"
      feature="bb.feature.instance-count"
      :description="instanceCountAttention"
    />
    <div class="px-5 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :environment="selectedEnvironment?.uid ?? String(UNKNOWN_ID)"
        :include-all="true"
        @update:environment="selectEnvironment"
      />
      <SearchBox
        v-model:value="state.searchText"
        :autofocus="true"
        :placeholder="$t('instance.search-instance-name')"
      />
    </div>
    <InstanceV1Table
      :instance-list="filteredInstanceV1List"
      :can-assign-license="subscriptionStore.currentPlan !== PlanType.FREE"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  EnvironmentTabFilter,
  InstanceV1Table,
  SearchBox,
} from "@/components/v2";
import {
  useUIStateStore,
  useSubscriptionV1Store,
  useEnvironmentV1Store,
  useEnvironmentV1List,
  useInstanceV1List,
  useInstanceV1Store,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { UNKNOWN_ID } from "../types";
import { sortInstanceV1ListByEnvironmentV1 } from "../utils";

interface LocalState {
  searchText: string;
}

const subscriptionStore = useSubscriptionV1Store();
const instanceV1Store = useInstanceV1Store();
const uiStateStore = useUIStateStore();
const router = useRouter();
const { t } = useI18n();

const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const state = reactive<LocalState>({
  searchText: "",
});

const selectedEnvironment = computed(() => {
  const uid = router.currentRoute.value.query.environment as string;
  if (uid) return useEnvironmentV1Store().getEnvironmentByUID(uid);
  return undefined;
});

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("instance.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "instance.visit",
      newState: true,
    });
  }
});

const selectEnvironment = (uid: string | undefined) => {
  if (uid && uid !== String(UNKNOWN_ID)) {
    router.replace({
      name: "workspace.instance",
      query: { environment: uid },
    });
  } else {
    router.replace({ name: "workspace.instance" });
  }
};

const { instanceList: rawInstanceV1List } = useInstanceV1List(
  false /* !showDeleted */
);

const filteredInstanceV1List = computed(() => {
  let list = [...rawInstanceV1List.value];
  const environment = selectedEnvironment.value;
  if (environment && environment.uid !== String(UNKNOWN_ID)) {
    list = list.filter((instance) => instance.environment === environment.name);
  }
  const keyword = state.searchText.trim().toLowerCase();
  if (keyword) {
    list = list.filter((instance) =>
      instance.title.toLowerCase().includes(keyword)
    );
  }

  return sortInstanceV1ListByEnvironmentV1(list, environmentList.value);
});

const remainingInstanceCount = computed((): number => {
  if (subscriptionStore.currentPlan === PlanType.FREE) {
    return Math.max(
      0,
      subscriptionStore.instanceCountLimit -
        instanceV1Store.activeInstanceList.length
    );
  }

  return Math.max(
    0,
    subscriptionStore.instanceLicenseCount -
      instanceV1Store.activateInstanceCount
  );
});

const instanceCountAttention = computed((): string => {
  const upgrade = t("subscription.features.bb-feature-instance-count.upgrade");
  let status = "";

  switch (subscriptionStore.currentPlan) {
    case PlanType.FREE:
      if (remainingInstanceCount.value > 0) {
        status = t(
          "subscription.features.bb-feature-instance-count.remaining",
          {
            total: subscriptionStore.instanceCountLimit,
            count: remainingInstanceCount.value,
          }
        );
      } else {
        status = t("subscription.features.bb-feature-instance-count.runoutof", {
          total: subscriptionStore.instanceCountLimit,
        });
      }
      break;
    case PlanType.TEAM:
    case PlanType.ENTERPRISE:
      if (remainingInstanceCount.value > 0) {
        status = t(
          "subscription.features.bb-feature-instance-count.remaining",
          {
            total: subscriptionStore.instanceLicenseCount,
            count: remainingInstanceCount.value,
          }
        );
      } else {
        status = t("subscription.features.bb-feature-instance-count.runoutof", {
          total: subscriptionStore.instanceLicenseCount,
        });
      }
      break;
  }

  return `${status} ${upgrade}`;
});
</script>
