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

    <div class="space-y-2">
      <InstanceOperations :instance-list="selectedInstanceList" />
      <PagedInstanceTable
        session-key="bb.instance-table"
        :bordered="false"
        :filter="filter"
        :footer-class="'mx-4'"
        :selected-instance-names="Array.from(state.selectedInstance)"
        :on-click="showInstanceDetail"
        @update:selected-instance-names="
          (list) => (state.selectedInstance = new Set(list))
        "
      />
    </div>

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
import {
  Drawer,
  DrawerContent,
  PagedInstanceTable,
  InstanceOperations,
} from "@/components/v2";
import {
  useSubscriptionV1Store,
  useInstanceV1Store,
  useAppFeature,
  useActuatorV1Store,
} from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { isValidInstanceName } from "@/types";
import { engineFromJSON } from "@/types/proto/v1/common";
import type { Instance } from "@/types/proto/v1/instance_service";
import {
  type SearchParams,
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
  selectedInstance: Set<string>;
}

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();
const instanceV1Store = useInstanceV1Store();
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
  selectedInstance: new Set(),
});

const scopeOptions = useCommonSearchScopeOptions(["environment", "engine"]);

const selectedEnvironment = computed(() => {
  const environmentId = state.params.scopes.find(
    (scope) => scope.id === "environment"
  )?.value;
  if (!environmentId) {
    return;
  }
  return `${environmentNamePrefix}${environmentId}`;
});

const selectedEngines = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "engine")
    .map((scope) => engineFromJSON(scope.value));
});

const filter = computed(() => ({
  environment: selectedEnvironment.value,
  query: state.params.query,
  engines: selectedEngines.value,
}));

onMounted(async () => {
  if (route.hash === "#add") {
    state.detail.show = true;
    state.detail.instance = undefined;
    return;
  }

  const maybeInstanceName = route.hash.replace(/^#*/g, "");
  if (isValidInstanceName(maybeInstanceName)) {
    try {
      const instance =
        await instanceV1Store.getOrFetchInstanceByName(maybeInstanceName);
      if (instance) {
        state.detail.show = true;
        state.detail.instance = instance;
      }
    } finally {
      // ignore error
    }
  }
});

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

const remainingInstanceCount = computed((): number => {
  return Math.max(
    0,
    subscriptionStore.instanceCountLimit - actuatorStore.totalInstanceCount
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

const selectedInstanceList = computed(() => {
  return [...state.selectedInstance]
    .filter((instanceName) => isValidInstanceName(instanceName))
    .map((instanceName) => instanceV1Store.getInstanceByName(instanceName));
});
</script>

<style scoped lang="postcss">
.instance-detail-drawer :deep(.n-drawer-header__main) {
  @apply flex-1 flex items-center justify-between;
}
</style>
