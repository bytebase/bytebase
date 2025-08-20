<template>
  <div class="w-full flex flex-col gap-4">
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
          :buttons-class="buttonsClass"
          @delete="doDelete"
        />
      </NTabPane>
      <template #suffix>
        <div
          v-if="!state.reorder"
          class="flex items-center justify-end space-x-2 px-2 pb-1"
        >
          <NButton
            v-if="
              hasWorkspacePermissionV2('bb.settings.get') &&
              hasWorkspacePermissionV2('bb.settings.set')
            "
            @click="startReorder"
          >
            <template #icon>
              <ListOrderedIcon class="h-4 w-4" />
            </template>
            {{ $t("common.reorder") }}
          </NButton>
          <NButton
            v-if="hasWorkspacePermissionV2('bb.settings.set')"
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
import { create } from "@bufbuild/protobuf";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  PlusIcon,
  ListOrderedIcon,
} from "lucide-vue-next";
import { NTabs, NTabPane, NButton } from "naive-ui";
import { onMounted, computed, reactive, watch, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { BBTabItem } from "@/bbkit/types";
import {
  EnvironmentForm,
  Form as EnvironmentFormBody,
  Buttons as EnvironmentFormButtons,
} from "@/components/EnvironmentForm";
import { Drawer, DrawerContent } from "@/components/v2";
import { EnvironmentV1Name, MiniActionButton } from "@/components/v2";
import {
  useUIStateStore,
  useEnvironmentV1Store,
  useEnvironmentV1List,
  environmentNamePrefix,
  pushNotification,
} from "@/store";
import {
  usePolicyV1Store,
  getEmptyRolloutPolicy,
} from "@/store/modules/v1/policy";
import { formatEnvironmentName } from "@/types";
import type { Policy } from "@/types/proto-es/v1/org_policy_service_pb";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import { EnvironmentSetting_EnvironmentSchema } from "@/types/proto-es/v1/setting_service_pb";
import type { Environment } from "@/types/v1/environment";
import { arraySwap, hasWorkspacePermissionV2 } from "@/utils";
import { type VueClass } from "@/utils";
import EnvironmentDetail from "@/views/EnvironmentDetail.vue";

const DEFAULT_NEW_ROLLOUT_POLICY: Policy = getEmptyRolloutPolicy(
  "",
  PolicyResourceType.ENVIRONMENT
);

defineProps<{
  buttonsClass?: VueClass;
}>();

interface LocalState {
  selectedId: string;
  reorderedEnvironmentList: Environment[];
  showCreateModal: boolean;
  reorder: boolean;
}

const { t } = useI18n();
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
  if (environmentList.value.length <= 0) {
    return;
  }
  const target = router.currentRoute.value.hash.slice(1);
  for (let i = 0; i < environmentList.value.length; i++) {
    const id = environmentList.value[i].id;
    if (id === target) {
      selectEnvironment(i);
      return;
    }
  }
  selectEnvironment(0);
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
    selectEnvironmentOnHash();
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
      const id = item.id;
      return { title, id, data: item };
    });
  }
  return [];
});

const getEnvironmentCreate = () => {
  return {
    ...create(EnvironmentSetting_EnvironmentSchema, {}),
    order: 0,
  };
};

const createEnvironment = () => {
  stopReorder();
  state.showCreateModal = true;
};

const doCreate = async (params: {
  environment: Partial<Environment>;
  rolloutPolicy: Policy;
}) => {
  const { environment, rolloutPolicy } = params;
  const createdEnvironment = await environmentV1Store.createEnvironment({
    id: environment.id,
    title: environment.title,
    order: environmentList.value.length,
    color: environment.color,
  });
  await environmentV1Store.fetchEnvironments();

  const requests = [
    policyV1Store.upsertPolicy({
      parentPath: `${environmentNamePrefix}${createdEnvironment.id}`,
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
  environmentV1Store
    .reorderEnvironmentList(state.reorderedEnvironmentList)
    .then(() => {
      stopReorder();
    });
};

const doDelete = async (environment: Environment) => {
  await environmentV1Store.deleteEnvironment(
    formatEnvironmentName(environment.id)
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.deleted"),
  });
  if (environmentList.value.length > 0) {
    selectEnvironment(0);
  }
};

const selectEnvironment = (index: number) => {
  const id = environmentList.value[index].id;
  onTabChange(id);
};

const onTabChange = (id: string) => {
  state.selectedId = id;
  router.replace({
    name: router.currentRoute.value.name,
    hash: "#" + id,
  });
};

const renderTab = (env: Environment, index: number) => {
  const child = [
    h(EnvironmentV1Name, {
      environment: env,
      link: false,
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
  } else {
    child.unshift(h("span", { class: "text-opacity-60" }, `${index + 1}.`));
  }

  return h("div", { class: "flex items-center space-x-2 py-1" }, child);
};
</script>
