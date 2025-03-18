<template>
  <div class="w-full flex flex-col gap-4 py-4 overflow-y-auto">
    <FeatureAttention
      v-if="remainingInstanceCount <= 3"
      feature="bb.feature.instance-count"
      :description="instanceCountAttention"
    />
    <div class="px-4 flex items-center space-x-2">
      <AdvancedSearch
        v-model:params="state.params"
        :autofocus="false"
        :placeholder="$t('instance.filter-instance-name')"
        :scope-options="scopeOptions"
      />
      <NButton type="primary" @click="handleClickAddInstance">
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        <NEllipsis>
          {{ $t("quick-action.add-instance") }}
        </NEllipsis>
      </NButton>
    </div>
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
        v-if="!state.detail.instance"
        :instance="state.detail.instance"
        :hide-advanced-features="hideAdvancedFeatures"
        @dismiss="state.detail.show = false"
      >
        <DrawerContent
          :title="detailTitle"
          class="instance-detail-drawer"
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
      <DrawerContent
        v-else
        header-style="--n-header-padding: 0 24px;"
        body-style="--n-body-padding: 16px 24px 0;"
      >
        <template #header>
          <div class="h-[50px] flex">
            <div class="flex items-center gap-x-2 h-[50px]">
              <EngineIcon
                :engine="state.detail.instance.engine"
                custom-class="!h-6"
              />
              <span class="font-medium">{{
                instanceV1Name(state.detail.instance)
              }}</span>
            </div>
          </div>
        </template>
        <InstanceDetail
          :instance-id="extractInstanceResourceName(state.detail.instance.name)"
          :embedded="true"
          :hide-archive-restore="true"
          class="!px-0 !mb-0 w-[850px]"
        />
      </DrawerContent>
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
import { EngineIcon } from "@/components/Icon";
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
  useAppFeature,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import type { Instance } from "@/types/proto/v1/instance_service";
import { PlanType } from "@/types/proto/v1/subscription_service";
import {
  type SearchParams,
  sortInstanceV1ListByEnvironmentV1,
  extractEnvironmentResourceName,
  wrapRefAsPromise,
  extractInstanceResourceName,
  instanceV1Name,
} from "@/utils";
import InstanceDetail from "@/views/InstanceDetail.vue";

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
// Users are workspace admins in Bytebase Editor. So we don't need to check permission here.
const { instanceList, ready } = useInstanceV1List();
const hideAdvancedFeatures = useAppFeature(
  "bb.feature.sql-editor.hide-advance-instance-features"
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

const scopeOptions = useCommonSearchScopeOptions(["environment"]);

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
      const instance = instanceList.value.find(
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
  let list = [...instanceList.value];
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
    subscriptionStore.instanceCountLimit - instanceV1Store.instanceList.length
  );
});

const instanceCountAttention = computed((): string => {
  const upgrade = t(
    "dynamic.subscription.features.bb-feature-instance-count.upgrade"
  );
  let status = "";

  if (remainingInstanceCount.value > 0) {
    status = t(
      "dynamic.subscription.features.bb-feature-instance-count.remaining",
      {
        total: subscriptionStore.instanceCountLimit,
        count: remainingInstanceCount.value,
      }
    );
  } else {
    status = t(
      "dynamic.subscription.features.bb-feature-instance-count.runoutof",
      {
        total: subscriptionStore.instanceCountLimit,
      }
    );
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

<style scoped lang="postcss">
.instance-detail-drawer :deep(.n-drawer-header__main) {
  @apply flex-1 flex items-center justify-between;
}
</style>
