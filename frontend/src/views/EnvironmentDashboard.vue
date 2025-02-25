<template>
  <div class="w-full flex flex-col gap-4 px-2">
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
          :environment-name="item.id"
          @archive="doArchive"
        />
      </NTabPane>
      <template #suffix>
        <div
          v-if="!state.reorder"
          class="flex items-center justify-end space-x-2 px-2 pb-1"
        >
          <NButton
            v-if="
              hasWorkspacePermissionV2('bb.environments.list') &&
              hasWorkspacePermissionV2('bb.environments.update')
            "
            @click="startReorder"
          >
            <template #icon>
              <ListOrderedIcon class="h-4 w-4" />
            </template>
            {{ $t("common.reorder") }}
          </NButton>
          <NButton
            v-if="hasWorkspacePermissionV2('bb.environments.create')"
            type="primary"
            @click="createEnvironment"
          >
            <template #icon>
              <PlusIcon class="h-4 w-4" />
            </template>
            {{ $t("environment.create") }}
          </NButton>
        </div>
      </template>
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
      :environment="getEnvironmentCreate()"
      :rollout-policy="DEFAULT_NEW_ROLLOUT_POLICY"
      :environment-tier="defaultEnvironmentTier"
      @create="doCreate"
      @cancel="state.showCreateModal = false"
    >
      <DrawerContent :title="$t('environment.create')">
        <EnvironmentFormBody class="w-[36rem]" />
        <template #footer>
          <EnvironmentFormButtons />
        </template>
      </DrawerContent>
    </EnvironmentForm>
  </Drawer>
</template>

<script lang="ts" setup>
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  PlusIcon,
  ListOrderedIcon,
} from "lucide-vue-next";
import { NTabs, NTabPane, NButton } from "naive-ui";
import { onMounted, computed, reactive, watch, h } from "vue";
import { useRouter } from "vue-router";
import type { BBTabItem } from "@/bbkit/types";
import {
  EnvironmentForm,
  Form as EnvironmentFormBody,
  Buttons as EnvironmentFormButtons,
} from "@/components/EnvironmentForm";
import { Drawer, DrawerContent } from "@/components/v2";
import { EnvironmentV1Name, MiniActionButton } from "@/components/v2";
import { useBodyLayoutContext } from "@/layouts/common";
import { ENVIRONMENT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import {
  useUIStateStore,
  useEnvironmentV1Store,
  defaultEnvironmentTier,
  useEnvironmentV1List,
} from "@/store";
import {
  usePolicyV1Store,
  getEmptyRolloutPolicy,
} from "@/store/modules/v1/policy";
import { emptyEnvironment } from "@/types";
import type {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import type { Policy } from "@/types/proto/v1/org_policy_service";
import { PolicyResourceType } from "@/types/proto/v1/org_policy_service";
import {
  arraySwap,
  extractEnvironmentResourceName,
  hasWorkspacePermissionV2,
} from "@/utils";
import EnvironmentDetail from "@/views/EnvironmentDetail.vue";

const DEFAULT_NEW_ROLLOUT_POLICY: Policy = getEmptyRolloutPolicy(
  "",
  PolicyResourceType.ENVIRONMENT
);

interface LocalState {
  selectedId: string;
  reorderedEnvironmentList: Environment[];
  showCreateModal: boolean;
  reorder: boolean;
}

const environmentV1Store = useEnvironmentV1Store();
const uiStateStore = useUIStateStore();
const policyV1Store = usePolicyV1Store();
const router = useRouter();

const { overrideMainContainerClass } = useBodyLayoutContext();
overrideMainContainerClass("!pb-0");

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

const doCreate = async (params: {
  environment: Partial<Environment>;
  rolloutPolicy: Policy;
  environmentTier: EnvironmentTier;
}) => {
  const { environment, rolloutPolicy, environmentTier } = params;
  const createdEnvironment = await environmentV1Store.createEnvironment({
    name: environment.name,
    title: environment.title,
    order: environmentList.value.length,
    color: environment.color,
    tier: environmentTier,
  });
  await environmentV1Store.fetchEnvironments();

  const requests = [
    policyV1Store.upsertPolicy({
      parentPath: createdEnvironment.name,
      policy: rolloutPolicy,
    }),
  ];
  await Promise.all(requests);
  state.showCreateModal = false;
  selectEnvironment(createdEnvironment.order);
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
          {
            default: () => h(ChevronLeftIcon, { class: "w-4 h-4" }),
          }
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
          {
            default: () => h(ChevronRightIcon, { class: "w-4 h-4" }),
          }
        )
      );
    }
  }

  return h("div", { class: "flex items-center space-x-2 py-1" }, child);
};
</script>
