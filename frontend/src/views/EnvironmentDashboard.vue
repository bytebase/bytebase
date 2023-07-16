<template>
  <div class="h-full overflow-hidden flex flex-col">
    <BBTab
      :tab-item-list="tabItemList"
      :selected-index="state.selectedIndex"
      :reorder-model="state.reorder ? 'ALWAYS' : 'NEVER'"
      @reorder-index="reorderEnvironment"
      @select-index="selectEnvironment"
    >
      <template
        #item="{ item }: { item: BBTabItem<Environment>, index: number }"
      >
        <div class="flex items-center">
          {{ item.title }}
          <ProductionEnvironmentV1Icon :environment="item.data!" class="ml-1" />
        </div>
      </template>

      <BBTabPanel
        v-for="(env, index) in environmentList"
        :key="env.uid"
        :active="index == state.selectedIndex"
        class="flex-1 overflow-y-scroll"
      >
        <div v-if="state.reorder" class="flex justify-center pt-5">
          <button
            type="button"
            class="btn-normal py-2 px-4"
            @click.prevent="discardReorder"
          >
            {{ $t("common.cancel") }}
          </button>
          <button
            type="submit"
            class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
            :disabled="!orderChanged"
            @click.prevent="doReorder"
          >
            {{ $t("common.apply") }}
          </button>
        </div>
        <EnvironmentDetail
          v-else
          :environment-slug="environmentV1Slug(env)"
          @archive="doArchive"
        />
      </BBTabPanel>
    </BBTab>
  </div>

  <Drawer v-model:show="state.showCreateModal">
    <EnvironmentForm
      :create="true"
      :drawer="true"
      :environment="getEnvironmentCreate()"
      :approval-policy="(DEFAULT_NEW_APPROVAL_POLICY as any)"
      :backup-policy="(DEFAULT_NEW_BACKUP_PLAN_POLICY as any)"
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
import { onMounted, computed, reactive, watch } from "vue";
import { useRouter } from "vue-router";
import { arraySwap, environmentV1Slug } from "../utils";
import EnvironmentDetail from "../views/EnvironmentDetail.vue";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import type { BBTabItem } from "../bbkit/types";
import {
  useRegisterCommand,
  useUIStateStore,
  hasFeature,
  useEnvironmentV1Store,
  defaultEnvironmentTier,
  useEnvironmentV1List,
} from "@/store";
import { Drawer, ProductionEnvironmentV1Icon } from "@/components/v2";
import {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import {
  usePolicyV1Store,
  defaultBackupSchedule,
  defaultApprovalStrategy,
  getDefaultBackupPlanPolicy,
  getDefaultDeploymentApprovalPolicy,
} from "@/store/modules/v1/policy";
import {
  Policy,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { emptyEnvironment } from "@/types";

// The default value should be consistent with the GetDefaultPolicy from the backend.
const DEFAULT_NEW_APPROVAL_POLICY: Policy = getDefaultDeploymentApprovalPolicy(
  "",
  PolicyResourceType.ENVIRONMENT
);

// The default value should be consistent with the GetDefaultPolicy from the backend.
const DEFAULT_NEW_BACKUP_PLAN_POLICY: Policy = getDefaultBackupPlanPolicy(
  "",
  PolicyResourceType.ENVIRONMENT
);

interface LocalState {
  reorderedEnvironmentList: Environment[];
  selectedIndex: number;
  showCreateModal: boolean;
  reorder: boolean;
  missingRequiredFeature?:
    | "bb.feature.approval-policy"
    | "bb.feature.backup-policy"
    | "bb.feature.environment-tier-policy";
}

const environmentV1Store = useEnvironmentV1Store();
const uiStateStore = useUIStateStore();
const policyV1Store = usePolicyV1Store();
const router = useRouter();

const state = reactive<LocalState>({
  reorderedEnvironmentList: [],
  selectedIndex: -1,
  showCreateModal: false,
  reorder: false,
});

const selectEnvironmentOnHash = () => {
  if (environmentList.value.length > 0) {
    if (router.currentRoute.value.hash) {
      for (let i = 0; i < environmentList.value.length; i++) {
        if (
          environmentList.value[i].uid ===
          router.currentRoute.value.hash.slice(1)
        ) {
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
    if (router.currentRoute.value.name == "workspace.environment") {
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
      const id = item.uid;
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
  approvalPolicy: Policy,
  backupPolicy: Policy,
  environmentTier: EnvironmentTier
) => {
  if (
    approvalPolicy.deploymentApprovalPolicy?.defaultStrategy !==
      defaultApprovalStrategy &&
    !hasFeature("bb.feature.approval-policy")
  ) {
    state.missingRequiredFeature = "bb.feature.approval-policy";
    return;
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
      policy: approvalPolicy,
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
    if (state.reorderedEnvironmentList[i].uid != environmentList.value[i].uid) {
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
  state.selectedIndex = index;
  router.replace({
    name: "workspace.environment",
    hash: "#" + environmentList.value[index].uid,
  });
};
</script>
