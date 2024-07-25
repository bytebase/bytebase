<template>
  <div class="w-full flex flex-col gap-4 py-4 px-2 overflow-y-auto">
    <div class="grid grid-cols-3 gap-x-2 gap-y-4 md:inline-flex items-stretch">
      <NButton @click="handleClickAddInstance">
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        <NEllipsis>
          {{ $t("quick-action.add-instance") }}
        </NEllipsis>
      </NButton>
    </div>

    <FeatureAttention
      v-if="remainingInstanceCount <= 3"
      feature="bb.feature.instance-count"
      :description="instanceCountAttention"
    />
    <AdvancedSearch
      v-model:params="state.params"
      :autofocus="false"
      :placeholder="$t('instance.filter-instance-name')"
      :scope-options="scopeOptions"
    />
    <InstanceV1Table
      :bordered="false"
      :loading="!ready"
      :instance-list="filteredInstanceV1List"
      :can-assign-license="subscriptionStore.currentPlan !== PlanType.FREE"
      :on-click="showInstanceDetail"
    />

    <Drawer
      v-model:show="state.detail.show"
      :close-on-esc="!!state.detail.instance"
      :mask-closable="!!state.detail.instance"
    >
      <InstanceForm
        :instance="state.detail.instance"
        @dismiss="state.detail.show = false"
      >
        <DrawerContent
          :title="detailTitle"
          body-content-class="flex flex-col gap-2 overflow-hidden"
        >
          <InstanceFormBody
            :hide-archive-restore="true"
            class="flex-1 overflow-auto"
          />
          <InstanceFormButtons
            class="border-t border-block-border pt-4 pb-0"
            :on-created="(instance) => (state.detail.instance = instance)"
            :on-updated="(instance) => (state.detail.instance = instance)"
          />
        </DrawerContent>
      </InstanceForm>
    </Drawer>
  </div>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton, NEllipsis } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { FeatureAttention } from "@/components/FeatureGuard";
import {
  InstanceForm,
  Form as InstanceFormBody,
  Buttons as InstanceFormButtons,
} from "@/components/InstanceForm";
import { Drawer, DrawerContent, InstanceV1Table } from "@/components/v2";
import {
  useSubscriptionV1Store,
  useEnvironmentV1List,
  useInstanceV1List,
  useInstanceV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import type { Instance } from "@/types/proto/v1/instance_service";
import { PlanType } from "@/types/proto/v1/subscription_service";
import {
  type SearchParams,
  sortInstanceV1ListByEnvironmentV1,
  extractEnvironmentResourceName,
  wrapRefAsPromise,
} from "@/utils";

interface LocalState {
  params: SearchParams;
  detail: {
    show: boolean;
    instance: Instance | undefined;
  };
}

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const subscriptionStore = useSubscriptionV1Store();
const instanceV1Store = useInstanceV1Store();
const environmentList = useEnvironmentV1List();
const { instanceList: rawInstanceV1List, ready } = useInstanceV1List(
  /* showDeleted */ false,
  /* forceUpdate */ true
);

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  detail: {
    show: false,
    instance: undefined,
  },
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  ["environment"]
);

const selectedEnvironment = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "environment")?.value ??
    `${UNKNOWN_ID}`
  );
});

onMounted(() => {
  if (route.hash === "#add") {
    state.detail.show = true;
    state.detail.instance = undefined;
  }
  wrapRefAsPromise(ready, true).then(() => {
    const maybeInstanceName = route.hash.replace(/^#*/g, "");
    if (maybeInstanceName) {
      const instance = rawInstanceV1List.value.find(
        (inst) => inst.name === maybeInstanceName
      );
      if (instance) {
        state.detail.show = true;
        state.detail.instance = instance;
      }
    }

    watch(
      [() => state.detail.show, () => state.detail.instance?.name],
      ([show, instanceName]) => {
        if (show) {
          router.replace({ hash: instanceName ? `#${instanceName}` : "#add" });
        } else {
          router.replace({ hash: "" });
        }
      }
    );
  });
});

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

const detailTitle = computed(() => {
  return state.detail.instance
    ? `${t("common.instance")} - ${state.detail.instance.title}`
    : t("quick-action.add-instance");
});

const handleClickAddInstance = () => {
  state.detail.show = true;
  state.detail.instance = undefined;
};

const showInstanceDetail = (instance: Instance) => {
  state.detail.show = true;
  state.detail.instance = instance;
};
</script>
