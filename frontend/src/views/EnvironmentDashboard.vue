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
      :environment="DEFAULT_NEW_ENVIRONMENT"
      :approval-policy="(DEFAULT_NEW_APPROVAL_POLICY as any)"
      :backup-policy="(DEFAULT_NEW_BACKUP_PLAN_POLICY as any)"
      :environment-tier-policy="(DEFAULT_NEW_ENVIRONMENT_TIER_POLICY as any)"
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
import { array_swap } from "../utils";
import EnvironmentDetail from "../views/EnvironmentDetail.vue";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import type {
  Environment,
  EnvironmentCreate,
  Policy,
  PolicyUpsert,
  PipelineApprovalPolicyPayload,
  BackupPlanPolicyPayload,
  EnvironmentTierPolicyPayload,
} from "../types";
import {
  DefaultApprovalPolicy,
  DefaultSchedulePolicy,
  DefaultEnvironmentTier,
} from "../types";
import type { BBTabItem } from "../bbkit/types";
import {
  useRegisterCommand,
  useUIStateStore,
  hasFeature,
  usePolicyStore,
  useEnvironmentStore,
  useEnvironmentList,
} from "@/store";
import ProductionEnvironmentIcon from "../components/Environment/ProductionEnvironmentIcon.vue";

const DEFAULT_NEW_ENVIRONMENT: EnvironmentCreate = {
  name: "New Env",
};

// The default value should be consistent with the GetDefaultPolicy from the backend.
const DEFAULT_NEW_APPROVAL_POLICY: PolicyUpsert = {
  payload: {
    value: DefaultApprovalPolicy,
    assigneeGroupList: [],
  },
};

// The default value should be consistent with the GetDefaultPolicy from the backend.
const DEFAULT_NEW_BACKUP_PLAN_POLICY: PolicyUpsert = {
  payload: {
    schedule: DefaultSchedulePolicy,
    retentionPeriodTs: 0,
  },
};

const DEFAULT_NEW_ENVIRONMENT_TIER_POLICY: PolicyUpsert = {
  payload: {
    environmentTier: DefaultEnvironmentTier,
  },
};

interface LocalState {
  reorderedEnvironmentList: Environment[];
  selectedIndex: number;
  showCreateModal: boolean;
  reorder: boolean;
  missingRequiredFeature?:
    | "bb.feature.approval-policy"
    | "bb.feature.backup-policy";
}

const environmentStore = useEnvironmentStore();
const uiStateStore = useUIStateStore();
const policyStore = usePolicyStore();
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

const createEnvironment = () => {
  stopReorder();
  state.showCreateModal = true;
};

const doCreate = (
  newEnvironment: EnvironmentCreate,
  approvalPolicy: Policy,
  backupPolicy: Policy,
  environmentTierPolicy: Policy
) => {
  if (
    (approvalPolicy.payload as PipelineApprovalPolicyPayload).value ===
      "MANUAL_APPROVAL_NEVER" &&
    !hasFeature("bb.feature.approval-policy")
  ) {
    state.missingRequiredFeature = "bb.feature.approval-policy";
    return;
  }
  if (
    (backupPolicy.payload as BackupPlanPolicyPayload).schedule !==
      DefaultSchedulePolicy &&
    !hasFeature("bb.feature.backup-policy")
  ) {
    state.missingRequiredFeature = "bb.feature.backup-policy";
    return;
  }

  environmentStore
    .createEnvironment(newEnvironment)
    .then((environment: Environment) => {
      const requests = [
        policyStore.upsertPolicyByEnvironmentAndType({
          environmentId: environment.id,
          type: "bb.policy.pipeline-approval",
          policyUpsert: { payload: approvalPolicy.payload },
        }),
        policyStore.upsertPolicyByEnvironmentAndType({
          environmentId: environment.id,
          type: "bb.policy.backup-plan",
          policyUpsert: { payload: backupPolicy.payload },
        }),
        policyStore.upsertPolicyByEnvironmentAndType({
          environmentId: environment.id,
          type: "bb.policy.environment-tier",
          policyUpsert: { payload: environmentTierPolicy.payload },
        }),
      ];
      // Also upsert the environment's access-control policy
      const environmentTierPayload =
        environmentTierPolicy.payload as EnvironmentTierPolicyPayload;
      const disallowed = environmentTierPayload.environmentTier === "PROTECTED";
      requests.push(
        policyStore.upsertPolicyByEnvironmentAndType({
          environmentId: environment.id,
          type: "bb.policy.access-control",
          policyUpsert: {
            inheritFromParent: false,
            payload: {
              disallowRuleList: [
                {
                  fullDatabase: disallowed,
                },
              ],
            },
          },
        })
      );
      return Promise.all(requests);
    })
    .then(() => {
      state.showCreateModal = false;
      selectEnvironment(environmentList.value.length - 1);
    });
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
  array_swap(state.reorderedEnvironmentList, sourceIndex, targetIndex);
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
