<template>
  <div class="w-full h-full flex flex-col gap-4">
    <NTabs
      type="line"
      :bar-width="200"
      :value="state.selectedId"
      size="large"
      justify-content="start"
      class="h-full"
      tab-style="margin: 0 1rem; padding-left: 2rem; padding: 0 2.5rem 0.5rem 2.5rem;"
      @update:value="onTabChange"
    >
      <NTabPane
        v-for="(item, index) in tabItemList"
        :key="item.id"
        :name="item.id"
        :tab="() => renderTab(item.data!, index)"
        class="h-full"
      >
        <EnvironmentDetail
          ref="environmentDetailRefs"
          :environment-name="item.id"
          :buttons-class="buttonsClass"
          @delete="doDelete"
        />
      </NTabPane>
      <template #suffix>
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="['bb.settings.set']"
        >
          <div class="flex items-center justify-end gap-x-2 px-2 pb-1">
            <NButton
              :disabled="slotProps.disabled || environmentList.length <= 1"
              @click="startReorder"
            >
              <template #icon>
                <ListOrderedIcon class="h-4 w-4" />
              </template>
              {{ t("common.reorder") }}
            </NButton>
            <NButton
              type="primary"
              :disabled="slotProps.disabled"
              @click="createEnvironment"
            >
              <template #icon>
                <PlusIcon class="h-4 w-4" />
              </template>
              {{ t("environment.create") }}
            </NButton>
          </div>
        </PermissionGuardWrapper>
      </template>
    </NTabs>
  </div>

  <Drawer :show="state.showCreateModal" @close="state.showCreateModal = false">
    <EnvironmentForm
      :create="true"
      :environment="getEnvironmentCreate()"
      :rollout-policy="DEFAULT_NEW_ROLLOUT_POLICY"
      @create="doCreate"
      @cancel="state.showCreateModal = false"
    >
      <DrawerContent
        :title="t('environment.create')"
        class="w-xl max-w-[100vw]"
      >
        <EnvironmentFormBody />
        <template #footer>
          <EnvironmentFormButtons />
        </template>
      </DrawerContent>
    </EnvironmentForm>
  </Drawer>

  <Drawer v-model:show="state.reorder" :close-on-esc="true">
    <DrawerContent :title="t('environment.reorder')" class="w-120 max-w-[90vw]">
      <div>
        <Draggable
          v-model="state.reorderedEnvironmentList"
          item-key="id"
          animation="300"
        >
          <template #item="{ element, index }">
            <div
              :key="(element as Environment).id"
              class="flex items-center justify-between p-2 hover:bg-gray-100 rounded-xs cursor-grab"
            >
              <div class="flex items-center gap-x-2">
                <span class="textinfo"> {{ index + 1 }}.</span>
                <EnvironmentV1Name :environment="(element as Environment)" :link="false" />
              </div>
              <GripVerticalIcon class="w-5 h-5 text-gray-500" />
            </div>
          </template>
        </Draggable>
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="discardReorder">{{ t("common.cancel") }}</NButton>
          <NButton type="primary" :disabled="!orderChanged" @click="doReorder">
            {{ t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { GripVerticalIcon, ListOrderedIcon, PlusIcon } from "lucide-vue-next";
import { NButton, NTabPane, NTabs } from "naive-ui";
import { computed, h, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import Draggable from "vuedraggable";
import {
  EnvironmentForm,
  Form as EnvironmentFormBody,
  Buttons as EnvironmentFormButtons,
} from "@/components/EnvironmentForm";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { Drawer, DrawerContent, EnvironmentV1Name } from "@/components/v2";
import { useRouteChangeGuard } from "@/composables/useRouteChangeGuard";
import {
  environmentNamePrefix,
  pushNotification,
  useEnvironmentV1List,
  useEnvironmentV1Store,
  useUIStateStore,
} from "@/store";
import {
  getEmptyRolloutPolicy,
  usePolicyV1Store,
} from "@/store/modules/v1/policy";
import { formatEnvironmentName } from "@/types";
import type { Policy } from "@/types/proto-es/v1/org_policy_service_pb";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import { EnvironmentSetting_EnvironmentSchema } from "@/types/proto-es/v1/setting_service_pb";
import type { Environment } from "@/types/v1/environment";
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
const environmentList = useEnvironmentV1List();
const environmentDetailRefs = ref<InstanceType<typeof EnvironmentDetail>[]>([]);

const state = reactive<LocalState>({
  selectedId: "",
  reorderedEnvironmentList: [],
  showCreateModal: false,
  reorder: false,
});

const selectEnvironment = (index: number) => {
  const id = environmentList.value[index].id;
  onTabChange(id);
};

useRouteChangeGuard(
  computed(() => environmentDetailRefs.value[0]?.isEditing ?? false)
);

const onTabChange = (id: string) => {
  // The NTabPane only render the selected environment,
  // so we only need to check the 1st environmentDetailRefs
  if (environmentDetailRefs.value[0]?.isEditing) {
    if (!window.confirm(t("common.leave-without-saving"))) {
      return;
    }
  }
  state.selectedId = id;
  router.replace({
    name: router.currentRoute.value.name,
    hash: "#" + id,
  });
};

const selectEnvironmentOnHash = (target: string) => {
  if (environmentList.value.length <= 0) {
    return;
  }
  const index = Math.max(
    0,
    environmentList.value.findIndex((env) => env.id === target)
  );
  selectEnvironment(index);
};

onMounted(() => {
  if (!uiStateStore.getIntroStateByKey("environment.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "environment.visit",
      newState: true,
    });
  }
});

watch(
  () => router.currentRoute.value.hash,
  (hash) => {
    selectEnvironmentOnHash(hash.slice(1));
  },
  { immediate: true }
);

const tabItemList = computed(() => {
  return environmentList.value.map((item, index: number) => {
    const title = `${index + 1}. ${item.title}`;
    const id = item.id;
    return { title, id, data: item };
  });
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
    tags: environment.tags,
  });
  await environmentV1Store.fetchEnvironments();

  // Only persist rollout policy if user customized it from the defaults
  // Otherwise, let the backend return defaults dynamically via GetPolicy API
  const isCustomized = !isEqual(rolloutPolicy, DEFAULT_NEW_ROLLOUT_POLICY);
  if (isCustomized) {
    await policyV1Store.upsertPolicy({
      parentPath: `${environmentNamePrefix}${createdEnvironment.id}`,
      policy: rolloutPolicy,
    });
  }

  state.showCreateModal = false;
  selectEnvironment(createdEnvironment.order);
};

const startReorder = () => {
  state.reorderedEnvironmentList = [...environmentList.value];
  state.reorder = true;
};

const stopReorder = () => {
  state.reorder = false;
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
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
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

const renderTab = (env: Environment, index: number) => {
  return h("div", { class: "flex items-center gap-x-2 py-1" }, [
    h("span", { class: "text-opacity-60" }, `${index + 1}.`),
    h(EnvironmentV1Name, {
      environment: env,
      link: false,
    }),
  ]);
};
</script>
