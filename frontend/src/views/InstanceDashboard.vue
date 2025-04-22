<template>
  <div class="flex flex-col space-y-4">
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
      <NButton
        v-if="hasWorkspacePermissionV2('bb.instances.create')"
        type="primary"
        @click="showCreateInstanceDrawer"
      >
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        {{ $t("quick-action.add-instance") }}
      </NButton>
    </div>
    <div class="space-y-2">
      <InstanceOperations :instance-list="selectedInstanceList" />
      <PagedInstanceTable
        session-key="bb.instance-table"
        :bordered="false"
        :filter="filter"
        :footer-class="'mx-4'"
        :on-click="onRowClick"
        :selected-instance-names="Array.from(state.selectedInstance)"
        @update:selected-instance-names="
          (list) => (state.selectedInstance = new Set(list))
        "
      />
    </div>
  </div>
  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="state.showCreateDrawer"
    @close="state.showCreateDrawer = false"
  >
    <InstanceForm
      :hide-advanced-features="hideAdvancedFeatures"
      :drawer="true"
      @dismiss="state.showCreateDrawer = false"
    >
      <DrawerContent :title="$t('quick-action.add-instance')">
        <InstanceFormBody />
        <template #footer>
          <InstanceFormButtons :on-created="onInstanceCreated" />
        </template>
      </DrawerContent>
    </InstanceForm>
  </Drawer>

  <FeatureModal
    :open="state.showFeatureModal"
    :feature="'bb.feature.instance-count'"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="tsx" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import AdvancedSearch from "@/components/AdvancedSearch";
import type { ScopeOption } from "@/components/AdvancedSearch/types";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { FeatureAttention, FeatureModal } from "@/components/FeatureGuard";
import {
  InstanceForm,
  Form as InstanceFormBody,
  Buttons as InstanceFormButtons,
} from "@/components/InstanceForm/";
import { InstanceOperations, PagedInstanceTable } from "@/components/v2";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  useAppFeature,
  useUIStateStore,
  useSubscriptionV1Store,
  useInstanceV1Store,
  useActuatorV1Store,
} from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { isValidInstanceName } from "@/types";
import { engineFromJSON } from "@/types/proto/v1/common";
import type { Instance } from "@/types/proto/v1/instance_service";
import { type SearchParams, hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  params: SearchParams;
  showCreateDrawer: boolean;
  showFeatureModal: boolean;
  selectedInstance: Set<string>;
}

const props = defineProps<{
  onRowClick?: (instance: Instance) => void;
}>();

const { t } = useI18n();
const subscriptionStore = useSubscriptionV1Store();
const instanceV1Store = useInstanceV1Store();
const uiStateStore = useUIStateStore();
const actuatorStore = useActuatorV1Store();
const router = useRouter();

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
  showFeatureModal: false,
  selectedInstance: new Set(),
});

const hideAdvancedFeatures = useAppFeature(
  "bb.feature.sql-editor.hide-advance-instance-features"
);

const onInstanceCreated = (instance: Instance) => {
  if (props.onRowClick) {
    return props.onRowClick(instance);
  }
  router.push(`/${instance.name}`);
  state.showCreateDrawer = false;
};

const selectedEnvironment = computed(() => {
  const environmentId = state.params.scopes.find(
    (scope) => scope.id === "environment"
  )?.value;
  if (!environmentId) {
    return;
  }
  return `${environmentNamePrefix}${environmentId}`;
});

const selectedHost = computed(() => {
  return state.params.scopes.find((scope) => scope.id === "host")?.value ?? "";
});

const selectedPort = computed(() => {
  return state.params.scopes.find((scope) => scope.id === "port")?.value ?? "";
});

const selectedEngines = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "engine")
    .map((scope) => engineFromJSON(scope.value));
});

const filter = computed(() => ({
  environment: selectedEnvironment.value,
  host: selectedHost.value,
  port: selectedPort.value,
  query: state.params.query,
  engines: selectedEngines.value,
}));

const showCreateInstanceDrawer = () => {
  if (
    subscriptionStore.instanceCountLimit <= actuatorStore.activatedInstanceCount
  ) {
    state.showFeatureModal = true;
    return;
  }
  state.showCreateDrawer = true;
};

const scopeOptions = computed((): ScopeOption[] => {
  return [
    ...useCommonSearchScopeOptions(["environment", "engine"]).value,
    {
      id: "host",
      title: t("instance.advanced-search.scope.host.title"),
      description: t("instance.advanced-search.scope.host.description"),
      options: [],
    },
    {
      id: "port",
      title: t("instance.advanced-search.scope.port.title"),
      description: t("instance.advanced-search.scope.port.description"),
      options: [],
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

const selectedInstanceList = computed(() => {
  return [...state.selectedInstance]
    .filter((instanceName) => isValidInstanceName(instanceName))
    .map((instanceName) => instanceV1Store.getInstanceByName(instanceName));
});
</script>
