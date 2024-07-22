<template>
  <div class="w-full flex flex-col gap-4 pt-4 px-2 overflow-hidden relative">
    <div class="grid grid-cols-3 gap-x-2 gap-y-4 md:inline-flex items-stretch">
      <NButton @click="handleClickCreateEnvironment">
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        <NEllipsis>
          {{ $t("environment.create") }}
        </NEllipsis>
      </NButton>
    </div>

    <div class="flex-1 overflow-y-auto">
      <NTabs
        type="line"
        :bar-width="200"
        :value="state.selectedId"
        size="large"
        justify-content="start"
        tab-style="margin: 0; padding: 0 2.5rem 0.5rem 2rem;"
        @update:value="onTabChange"
      >
        <NTabPane
          v-for="(item, index) in tabItemList"
          :key="item.id"
          :name="item.id"
          :tab="() => renderTab(item.data, index)"
        >
          <EnvironmentDetail
            :environment-name="item.id"
            :simple="true"
            :hide-archive-restore="true"
            body-class="!px-0"
            buttons-class="!absolute left-2 right-2"
            @archive="doArchive"
          />
        </NTabPane>
      </NTabs>
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
        <EnvironmentFormBody
          :simple="true"
          :hide-archive-restore="true"
          class="w-[36rem]"
        />
        <template #footer>
          <EnvironmentFormButtons />
        </template>
      </DrawerContent>
    </EnvironmentForm>
  </Drawer>

  <FeatureModal
    :open="state.missingRequiredFeature != undefined"
    :feature="state.missingRequiredFeature"
    @cancel="state.missingRequiredFeature = undefined"
  />
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NTabs, NTabPane, NButton, NEllipsis } from "naive-ui";
import { onMounted, computed, reactive, watch, h } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { BBTabItem } from "@/bbkit/types";
import {
  EnvironmentForm,
  Form as EnvironmentFormBody,
  Buttons as EnvironmentFormButtons,
} from "@/components/EnvironmentForm";
import { FeatureModal } from "@/components/FeatureGuard";
import { Drawer, DrawerContent } from "@/components/v2";
import { EnvironmentV1Name } from "@/components/v2";
import { SQL_EDITOR_SETTING_ENVIRONMENT_MODULE } from "@/router/sqlEditor";
import {
  hasFeature,
  useEnvironmentV1Store,
  defaultEnvironmentTier,
  useEnvironmentV1List,
} from "@/store";
import {
  usePolicyV1Store,
  getEmptyRolloutPolicy,
} from "@/store/modules/v1/policy";
import { VirtualRoleType, emptyEnvironment } from "@/types";
import type {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import type { Policy } from "@/types/proto/v1/org_policy_service";
import { PolicyResourceType } from "@/types/proto/v1/org_policy_service";
import { extractEnvironmentResourceName } from "@/utils";
import EnvironmentDetail from "@/views/EnvironmentDetail.vue";

const DEFAULT_NEW_ROLLOUT_POLICY: Policy = getEmptyRolloutPolicy(
  "",
  PolicyResourceType.ENVIRONMENT
);

interface LocalState {
  selectedId: string;
  showCreateModal: boolean;
  missingRequiredFeature?:
    | "bb.feature.approval-policy"
    | "bb.feature.custom-approval"
    | "bb.feature.environment-tier-policy";
}

const environmentV1Store = useEnvironmentV1Store();
const policyV1Store = usePolicyV1Store();
const route = useRoute();
const router = useRouter();

const state = reactive<LocalState>({
  selectedId: "",
  showCreateModal: false,
});

const selectEnvironmentOnHash = () => {
  if (environmentList.value.length > 0) {
    if (route.hash) {
      for (let i = 0; i < environmentList.value.length; i++) {
        const id = extractEnvironmentResourceName(
          environmentList.value[i].name
        );
        if (id === route.hash.slice(1)) {
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
});

watch(
  () => route.hash,
  () => {
    if (route.name == SQL_EDITOR_SETTING_ENVIRONMENT_MODULE) {
      selectEnvironmentOnHash();
    }
  }
);

const environmentList = useEnvironmentV1List();

const tabItemList = computed((): BBTabItem[] => {
  if (environmentList.value) {
    const list = environmentList.value;
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

const handleClickCreateEnvironment = () => {
  state.showCreateModal = true;
};

const doCreate = async (params: {
  environment: Partial<Environment>;
  rolloutPolicy: Policy;
  environmentTier: EnvironmentTier;
}) => {
  const { environment, rolloutPolicy, environmentTier } = params;
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

  const createdEnvironment = await environmentV1Store.createEnvironment({
    name: environment.name,
    title: environment.title,
    order: environmentList.value.length,
    tier: environmentTier,
  });
  await environmentV1Store.fetchEnvironments();

  const requests = [
    policyV1Store.upsertPolicy({
      parentPath: createdEnvironment.name,
      updateMask: ["payload"],
      policy: rolloutPolicy,
    }),
  ];
  await Promise.all(requests);
  state.showCreateModal = false;
  selectEnvironment(createdEnvironment.order);
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
    hash: "#" + id,
  });
};

const renderTab = (env: Environment, index: number) => {
  const child = [
    h(EnvironmentV1Name, {
      environment: env,
      link: false,
      prefix: `${index + 1}.`,
    }),
  ];
  return h("div", { class: "flex items-center space-x-2" }, child);
};
</script>
