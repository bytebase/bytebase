<template>
  <div>
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
          <ProductionEnvironmentIcon :environment="item.data!" class="ml-1" />
        </div>
      </template>

      <BBTabPanel
        v-for="(item, index) in environmentList"
        :key="item.id"
        :active="index == state.selectedIndex"
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
          :environment-slug="environmentSlug(item)"
          @archive="doArchive"
        />
      </BBTabPanel>
    </BBTab>
  </div>
  <BBModal
    v-if="state.showCreateModal"
    :title="$t('environment.create')"
    @close="state.showCreateModal = false"
  >
    <EnvironmentForm
      :create="true"
      :environment="getEnvironmentCreate()"
      :approval-policy="(DEFAULT_NEW_APPROVAL_POLICY as any)"
      :backup-policy="(DEFAULT_NEW_BACKUP_PLAN_POLICY as any)"
      :environment-tier="defaultEnvironmentTier"
      @create="doCreate"
      @cancel="state.showCreateModal = false"
    />
  </BBModal>

  <FeatureModal
    v-if="state.missingRequiredFeature != undefined"
    :feature="state.missingRequiredFeature"
    @cancel="state.missingRequiredFeature = undefined"
  />
</template>

<script lang="ts" setup>
import { onMounted, computed, reactive, watch } from "vue";
import { useRouter } from "vue-router";
import { arraySwap } from "../utils";
import EnvironmentDetail from "../views/EnvironmentDetail.vue";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import type { Environment, EnvironmentCreate } from "../types";
import type { BBTabItem } from "../bbkit/types";
import {
  useRegisterCommand,
  useUIStateStore,
  hasFeature,
  useEnvironmentStore,
  useEnvironmentList,
} from "@/store";
import ProductionEnvironmentIcon from "../components/Environment/ProductionEnvironmentIcon.vue";
import {
  useEnvironmentV1Store,
  defaultEnvironmentTier,
} from "@/store/modules/v1/environment";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import {
  usePolicyV1Store,
  defaultBackupSchedule,
  defaultApprovalStrategy,
  getDefaultBackupPlanPolicy,
  getDefaultDeploymentApprovalPolicy,
} from "@/store/modules/v1/policy";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";

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

const environmentStore = useEnvironmentStore();
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
          environmentList.value[i].id ===
          parseInt(router.currentRoute.value.hash.slice(1), 10)
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

const environmentList = useEnvironmentList();

const tabItemList = computed((): BBTabItem[] => {
  if (environmentList.value) {
    const list = state.reorder
      ? state.reorderedEnvironmentList
      : environmentList.value;
    return list.map((item: Environment, index: number): BBTabItem => {
      const title = `${index + 1}. ${item.name}`;
      const id = item.id.toString();
      return { title, id, data: item };
    });
  }
  return [];
});

const getEnvironmentCreate = (): EnvironmentCreate => {
  return {
    title: "",
    name: "",
  };
};

const createEnvironment = () => {
  stopReorder();
  state.showCreateModal = true;
};

const doCreate = async (
  newEnvironment: EnvironmentCreate,
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
  // After creating with v1 store, we need to fetch the latest data in old store.
  // TODO(steven): using grpc store.
  await environmentStore.fetchEnvironmentList();

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
    policyV1Store.upsertPolicy({
      parentPath: environment.name,
      updateMask: ["payload", "inherit_from_parent"],
      policy: {
        type: PolicyType.ACCESS_CONTROL,
        inheritFromParent: false,
        accessControlPolicy: {
          disallowRules: [
            {
              fullDatabase: environmentTier === EnvironmentTier.PROTECTED,
            },
          ],
        },
      },
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
    if (state.reorderedEnvironmentList[i].id != environmentList.value[i].id) {
      return true;
    }
  }
  return false;
});

const discardReorder = () => {
  stopReorder();
};

const doReorder = () => {
  environmentStore
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
    hash: "#" + environmentList.value[index].id,
  });
};
</script>
