<template>
  <div class="h-full flex flex-col pt-4">
    <NTabs
      type="line"
      :bar-width="200"
      :value="state.selectedId"
      size="large"
      justify-content="start"
      tab-style="margin: 0 1rem; padding-left: 2rem; padding: 0 2.5rem 0.5rem 2.5rem;"
      @update:value="onTabChange"
    >
      <NTabPane
        v-for="(item, index) in tabItemList"
        :key="item.id"
        :name="item.id"
        :tab="() => renderTab(item.data, index)"
      >
        <EnvironmentDetail
          v-if="!state.reorder"
          :environment-id="item.id"
          @archive="doArchive"
        />
      </NTabPane>
    </NTabs>
    <div v-if="state.reorder" class="flex justify-start pt-5 gap-x-3 px-5">
      <NButton @click.prevent="discardReorder">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton
        type="primary"
        :disabled="!orderChanged"
        @click.prevent="doReorder"
      >
        {{ $t("common.apply") }}
      </NButton>
    </div>
  </div>

  <Drawer v-model:show="state.showCreateModal">
    <EnvironmentForm
      :create="true"
      :drawer="true"
      :environment="getEnvironmentCreate()"
      :rollout-policy="DEFAULT_NEW_ROLLOUT_POLICY"
      :backup-policy="DEFAULT_NEW_BACKUP_PLAN_POLICY"
      :environment-tier="defaultEnvironmentTier"
      @create="doCreate"
      @cancel="state.showCreateModal = false"
    />
  </Drawer>

  <FeatureModal
    :open="state.missingRequiredFeature != undefined"
    :feature="state.missingRequiredFeature"
    @cancel="state.missingRequiredFeature = undefined"
  />
</template>

<script lang="ts" setup>
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-vue-next";
import { NTabs, NTabPane } from "naive-ui";
import { onMounted, computed, reactive, watch, h } from "vue";
import { useRouter } from "vue-router";
import { Drawer } from "@/components/v2";
import { EnvironmentV1Name, MiniActionButton } from "@/components/v2";
import { ENVIRONMENT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/environmentV1";
import {
  useRegisterCommand,
  useUIStateStore,
  hasFeature,
  useEnvironmentV1Store,
  defaultEnvironmentTier,
  useEnvironmentV1List,
} from "@/store";
import {
  usePolicyV1Store,
  defaultBackupSchedule,
  getDefaultBackupPlanPolicy,
  getEmptyRolloutPolicy,
} from "@/store/modules/v1/policy";
import { VirtualRoleType, emptyEnvironment } from "@/types";
import {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import {
  Policy,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import type { BBTabItem } from "../bbkit/types";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import { arraySwap, extractEnvironmentResourceName } from "../utils";
import EnvironmentDetail from "../views/EnvironmentDetail.vue";

const DEFAULT_NEW_ROLLOUT_POLICY: Policy = getEmptyRolloutPolicy(
  "",
  PolicyResourceType.ENVIRONMENT
);

// The default value should be consistent with the GetDefaultPolicy from the backend.
const DEFAULT_NEW_BACKUP_PLAN_POLICY: Policy = getDefaultBackupPlanPolicy(
  "",
  PolicyResourceType.ENVIRONMENT
);

interface LocalState {
  selectedId: string;
  reorderedEnvironmentList: Environment[];
  showCreateModal: boolean;
  reorder: boolean;
  missingRequiredFeature?:
    | "bb.feature.approval-policy"
    | "bb.feature.custom-approval"
    | "bb.feature.backup-policy"
    | "bb.feature.environment-tier-policy";
}

const environmentV1Store = useEnvironmentV1Store();
const uiStateStore = useUIStateStore();
const policyV1Store = usePolicyV1Store();
const router = useRouter();

const state = reactive<LocalState>({
  selectedId: "",
  reorderedEnvironmentList: [],
  showCreateModal: false,
  reorder: false,
});

const selectEnvironmentOnHash = () => {
  if (environmentList.value.length > 0) {
    if (router.currentRoute.value.hash) {
      for (let i = 0; i < environmentList.value.length; i++) {
        const id = extractEnvironmentResourceName(
          environmentList.value[i].name
        );
        if (id === router.currentRoute.value.hash.slice(1)) {
          selectEnvironment(i);
          break;
        }
      }
    } else {
      selectEnvironment(0);
    }
  }
};

onMounted(() => {
  selectEnvironmentOnHash();

  if (!uiStateStore.getIntroStateByKey("environment.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "environment.visit",
      newState: true,
    });
  }
});

useRegisterCommand({
  id: "bb.environment.create",
  registerId: "environment.dashboard",
  run: () => {
    createEnvironment();
  },
});
useRegisterCommand({
  id: "bb.environment.reorder",
  registerId: "environment.dashboard",
  run: () => {
    startReorder();
  },
});

watch(
  () => router.currentRoute.value.hash,
  () => {
    if (router.currentRoute.value.name == ENVIRONMENT_V1_ROUTE_DASHBOARD) {
      selectEnvironmentOnHash();
    }
  }
);

const environmentList = useEnvironmentV1List();

const tabItemList = computed((): BBTabItem[] => {
  if (environmentList.value) {
    const list = state.reorder
      ? state.reorderedEnvironmentList
      : environmentList.value;
    return list.map((item, index: number): BBTabItem => {
      const title = `${index + 1}. ${item.title}`;
      const id = extractEnvironmentResourceName(item.name);
      return { title, id, data: item };
    });
  }
  return [];
});

const getEnvironmentCreate = () => {
  return emptyEnvironment();
};

const createEnvironment = () => {
  stopReorder();
  state.showCreateModal = true;
};

const doCreate = async (
  newEnvironment: Environment,
  rolloutPolicy: Policy,
  backupPolicy: Policy,
  environmentTier: EnvironmentTier
) => {
  const rp = rolloutPolicy.rolloutPolicy;
  if (rp?.automatic === false) {
    if (rp.issueRoles.includes(VirtualRoleType.LAST_APPROVER)) {
      if (!hasFeature("bb.feature.custom-approval")) {
        state.missingRequiredFeature = "bb.feature.custom-approval";
        return;
      }
    }
    if (!hasFeature("bb.feature.approval-policy")) {
      state.missingRequiredFeature = "bb.feature.approval-policy";
      return;
    }
  }
  if (
    backupPolicy.backupPlanPolicy?.schedule !== defaultBackupSchedule &&
    !hasFeature("bb.feature.backup-policy")
  ) {
    state.missingRequiredFeature = "bb.feature.backup-policy";
    return;
  }
  if (
    environmentTier !== defaultEnvironmentTier &&
    !hasFeature("bb.feature.backup-policy")
  ) {
    state.missingRequiredFeature = "bb.feature.environment-tier-policy";
    return;
  }

  const environment = await environmentV1Store.createEnvironment({
    name: newEnvironment.name,
    title: newEnvironment.title,
    order: environmentList.value.length,
    tier: environmentTier,
  });
  await environmentV1Store.fetchEnvironments();

  const requests = [
    policyV1Store.upsertPolicy({
      parentPath: environment.name,
      updateMask: ["payload"],
      policy: rolloutPolicy,
    }),
    policyV1Store.upsertPolicy({
      parentPath: environment.name,
      updateMask: ["payload"],
      policy: backupPolicy,
    }),
  ];
  await Promise.all(requests);
  state.showCreateModal = false;
  selectEnvironment(environment.order);
};

const startReorder = () => {
  state.reorderedEnvironmentList = [...environmentList.value];
  state.reorder = true;
};

const stopReorder = () => {
  state.reorder = false;
  state.reorderedEnvironmentList = [];
};

const reorderEnvironment = (sourceIndex: number, targetIndex: number) => {
  arraySwap(state.reorderedEnvironmentList, sourceIndex, targetIndex);
  selectEnvironment(targetIndex);
};

const orderChanged = computed(() => {
  for (let i = 0; i < state.reorderedEnvironmentList.length; i++) {
    if (
      state.reorderedEnvironmentList[i].name != environmentList.value[i].name
    ) {
      return true;
    }
  }
  return false;
});

const discardReorder = () => {
  stopReorder();
};

const doReorder = () => {
  environmentV1Store
    .reorderEnvironmentList(state.reorderedEnvironmentList)
    .then(() => {
      stopReorder();
    });
};

const doArchive = (/* environment: Environment */) => {
  if (environmentList.value.length > 0) {
    selectEnvironment(0);
  }
};

const selectEnvironment = (index: number) => {
  const id = extractEnvironmentResourceName(environmentList.value[index].name);
  onTabChange(id);
};

const onTabChange = (id: string) => {
  state.selectedId = id;
  router.replace({
    name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
    hash: "#" + id,
  });
};

const renderTab = (env: Environment, index: number) => {
  const child = [
    h(EnvironmentV1Name, {
      environment: env,
      link: false,
      prefix: state.reorder ? "" : `${index + 1}.`,
    }),
  ];
  if (state.reorder) {
    if (index > 0) {
      child.unshift(
        h(
          MiniActionButton,
          {
            onClick: () => reorderEnvironment(index, index - 1),
          },
          h(ChevronLeftIcon, { class: "w-4 h-4" })
        )
      );
    }
    if (index < tabItemList.value.length - 1) {
      child.push(
        h(
          MiniActionButton,
          {
            onClick: () => reorderEnvironment(index, index + 1),
          },
          h(ChevronRightIcon, { class: "w-4 h-4" })
        )
      );
    }
  }

  return h("div", { class: "flex items-center space-x-2" }, child);
};
</script>
