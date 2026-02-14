<template>
  <div class="py-4 flex flex-col">
    <div class="px-4 flex flex-col gap-y-2 pb-2">
      <BBAttention
        v-if="remainingInstanceCount <= 3"
        :type="'warning'"
        :title="$t('subscription.usage.instance-count.title')"
        :description="instanceCountAttention"
      />
      <div class="flex items-center gap-x-2">
        <AdvancedSearch
          v-model:params="state.params"
          :autofocus="false"
          :placeholder="$t('instance.filter-instance-name')"
          :scope-options="scopeOptions"
        />
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="['bb.instances.create']"
        >
          <NButton
            type="primary"
            :disabled="slotProps.disabled"
            @click="showCreateInstanceDrawer"
          >
            <template #icon>
              <PlusIcon class="h-4 w-4" />
            </template>
            {{ $t("quick-action.add-instance") }}
          </NButton>
        </PermissionGuardWrapper>
      </div>
    </div>
    <div>
      <InstanceOperations
        :instance-list="selectedInstanceList"
        @update="(instances) => pagedInstanceTableRef?.updateCache(instances)"
      />
      <PagedInstanceTable
        ref="pagedInstanceTableRef"
        session-key="bb.instance-table"
        :bordered="false"
        :filter="filter"
        :footer-class="'mx-4'"
        :on-click="onRowClick"
        :selected-instance-names="Array.from(state.selectedInstance)"
        :show-external-link="false"
        :show-actions="true"
        @update:selected-instance-names="
          (list: string[]) => (state.selectedInstance = new Set(list))
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
      :hide-advanced-features="false"
      :drawer="true"
      @dismiss="state.showCreateDrawer = false"
    >
      <DrawerContent
        :title="$t('quick-action.add-instance')"
        class="w-212.5 max-w-[100vw]"
      >
        <InstanceFormBody />
        <template #footer>
          <InstanceFormButtons :on-created="onInstanceCreated" />
        </template>
      </DrawerContent>
    </InstanceForm>
  </Drawer>
</template>

<script lang="tsx" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import AdvancedSearch from "@/components/AdvancedSearch";
import type { ScopeOption } from "@/components/AdvancedSearch/types";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import {
  InstanceForm,
  Form as InstanceFormBody,
  Buttons as InstanceFormButtons,
} from "@/components/InstanceForm/";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import {
  Drawer,
  DrawerContent,
  InstanceOperations,
  PagedInstanceTable,
} from "@/components/v2";
import {
  useActuatorV1Store,
  useInstanceV1Store,
  useSubscriptionV1Store,
  useUIStateStore,
} from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { isValidInstanceName } from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import {
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  getValueFromSearchParams,
  getValuesFromSearchParams,
  hasWorkspacePermissionV2,
  type SearchParams,
} from "@/utils";

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
const route = useRoute();
const router = useRouter();
const pagedInstanceTableRef = ref<InstanceType<typeof PagedInstanceTable>>();

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
  showFeatureModal: false,
  selectedInstance: new Set(),
});

const onInstanceCreated = (instance: Instance) => {
  if (props.onRowClick) {
    return props.onRowClick(instance);
  }
  router.push(`/${instance.name}`);
  state.showCreateDrawer = false;
};

const selectedEnvironment = computed(() => {
  return getValueFromSearchParams(
    state.params,
    "environment",
    environmentNamePrefix
  );
});

const selectedHost = computed(() => {
  return getValueFromSearchParams(state.params, "host");
});

const selectedPort = computed(() => {
  return getValueFromSearchParams(state.params, "port");
});

const selectedEngines = computed(() => {
  return getValuesFromSearchParams(state.params, "engine").map(
    (engine) => Engine[engine as keyof typeof Engine]
  );
});

const selectedLabels = computed(() => {
  return getValuesFromSearchParams(state.params, "label");
});

const selectedState = computed(() => {
  const stateValue = getValueFromSearchParams(state.params, "state");
  if (stateValue === "DELETED") return State.DELETED;
  if (stateValue === "ALL") return undefined; // undefined = show all
  return State.ACTIVE; // default
});

const filter = computed(() => ({
  environment: selectedEnvironment.value,
  host: selectedHost.value,
  port: selectedPort.value,
  query: state.params.query,
  engines: selectedEngines.value,
  labels: selectedLabels.value,
  state: selectedState.value,
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
  const baseOptions = [
    ...useCommonSearchScopeOptions(["environment", "engine", "label", "state"])
      .value,
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

  // If user doesn't have undelete permission, remove DELETED and ALL from state scope
  if (!hasWorkspacePermissionV2("bb.instances.undelete")) {
    return baseOptions.map((scope) => {
      if (scope.id === "state" && scope.options) {
        return {
          ...scope,
          options: scope.options.filter((opt) => opt.value === "ACTIVE"),
        };
      }
      return scope;
    });
  }

  return baseOptions;
});

onMounted(() => {
  // Migrate old URL format (?state=archived) to new format (?q=state:archived)
  const queryState = router.currentRoute.value.query.state as string;
  if (queryState === "archived" || queryState === "all") {
    const stateValue = queryState === "archived" ? "DELETED" : "ALL";
    router.replace({
      query: { q: `state:${stateValue}` },
    });
  }

  // Initialize params from URL query
  const queryString = route.query.q as string;
  if (queryString) {
    state.params = buildSearchParamsBySearchText(queryString);
  }

  if (!uiStateStore.getIntroStateByKey("instance.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "instance.visit",
      newState: true,
    });
  }
});

// Sync params to URL query when params change
watch(
  () => state.params,
  (params) => {
    const queryString = buildSearchTextBySearchParams(params);
    const currentQuery = route.query.q as string;

    // Only update URL if query string has actually changed
    if (queryString !== currentQuery) {
      router.replace({
        query: queryString ? { q: queryString } : {},
      });
    }
  },
  { deep: true }
);

const remainingInstanceCount = computed((): number => {
  return Math.max(
    0,
    subscriptionStore.instanceCountLimit - actuatorStore.totalInstanceCount
  );
});

const instanceCountAttention = computed((): string => {
  const upgrade = t("subscription.usage.instance-count.upgrade");
  let status = "";

  if (remainingInstanceCount.value > 0) {
    status = t("subscription.usage.instance-count.remaining", {
      total: subscriptionStore.instanceCountLimit,
      count: remainingInstanceCount.value,
    });
  } else {
    status = t("subscription.usage.instance-count.runoutof", {
      total: subscriptionStore.instanceCountLimit,
    });
  }

  return `${status} ${upgrade}`;
});

const selectedInstanceList = computed(() => {
  return [...state.selectedInstance]
    .filter((instanceName) => isValidInstanceName(instanceName))
    .map((instanceName) => instanceV1Store.getInstanceByName(instanceName));
});
</script>
