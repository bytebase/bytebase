<template>
  <div class="w-full space-y-4">
    <div class="flex flex-row items-center justify-end">
      <div v-if="state.reorderRules" class="flex items-center gap-x-2">
        <NButton
          :disabled="state.processing"
          @click="
            () => {
              state.reorderRules = false;
              updateList();
            }
          "
        >
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="state.processing"
          @click="onReorderSubmit"
        >
          {{ $t("common.confirm") }}
        </NButton>
      </div>
      <div v-else class="flex items-center gap-x-2">
        <NButton
          class="capitalize"
          :disabled="
            !hasPermission ||
            !hasSensitiveDataFeature ||
            state.maskingRuleItemList.length <= 1
          "
          @click="state.reorderRules = true"
        >
          <template #icon>
            <ListOrderedIcon class="h-4 w-4" />
          </template>
          {{ $t("settings.sensitive-data.global-rules.re-order") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!hasPermission || !hasSensitiveDataFeature"
          @click="addNewRule"
        >
          <template #icon>
            <PlusIcon class="h-4 w-4" />
          </template>
          {{ $t("common.add") }}
        </NButton>
      </div>
    </div>
    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.global-rules.description") }}
      <LearnMoreLink
        url="https://docs.bytebase.com/security/data-masking/overview/?source=console"
      />
    </div>
    <NEmpty
      class="py-12 border rounded"
      v-if="state.maskingRuleItemList.length === 0"
    />
    <div
      v-for="(item, index) in state.maskingRuleItemList"
      :key="item.rule.id"
      class="flex items-start gap-x-5"
    >
      <div
        v-if="
          item.mode === 'NORMAL' && hasPermission && hasSensitiveDataFeature
        "
      >
        <div v-if="state.reorderRules" class="pt-2 flex flex-col">
          <MiniActionButton v-if="index > 0" @click="onReorder(item, -1)">
            <ChevronUpIcon class="w-4 h-4" />
          </MiniActionButton>
          <MiniActionButton
            v-if="index !== state.maskingRuleItemList.length - 1"
            @click="onReorder(item, 1)"
          >
            <ChevronDownIcon class="w-4 h-4" />
          </MiniActionButton>
        </div>
        <div v-else class="pt-2">
          <MiniActionButton @click="onEdit(index)">
            <PencilIcon class="w-4 h-4" />
          </MiniActionButton>
        </div>
      </div>
      <div
        class="pb-5"
        :class="[
          'w-full',
          item.mode === 'NORMAL' ? '' : 'ml-[42px]',
          index === state.maskingRuleItemList.length - 1 ? '' : 'border-b',
        ]"
      >
        <MaskingRuleConfig
          :key="`expr-${item.rule.id}`"
          :index="index + 1"
          :disabled="state.processing"
          :masking-rule="item.rule"
          :readonly="item.mode === 'NORMAL' || state.reorderRules"
          :factor-list="factorList"
          :option-config-map="factorOptionsMap"
          :allow-delete="item.mode === 'EDIT'"
          @cancel="onCancel(index)"
          @delete="onRuleDelete(index)"
          @confirm="(rule: MaskingRulePolicy_MaskingRule) => onConfirm(rule)"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import {
  PlusIcon,
  PencilIcon,
  ChevronUpIcon,
  ChevronDownIcon,
  ListOrderedIcon,
} from "lucide-vue-next";
import { NButton, NEmpty, type SelectOption } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, nextTick, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  getEnvironmentIdOptions,
  getProjectIdOptions,
} from "@/components/CustomApproval/Settings/components/common";
import { type OptionConfig } from "@/components/ExprEditor/context";
import { useBodyLayoutContext } from "@/layouts/common";
import type { Factor } from "@/plugins/cel";
import {
  featureToRef,
  pushNotification,
  usePolicyV1Store,
  useProjectV1Store,
  useInstanceV1Store,
} from "@/store";
import type { Policy } from "@/types/proto-es/v1/org_policy_service_pb";
import type { MaskingRulePolicy_MaskingRule } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  MaskingRulePolicySchema,
  MaskingRulePolicy_MaskingRuleSchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  arraySwap,
  hasWorkspacePermissionV2,
  getDefaultPagination,
} from "@/utils";
import LearnMoreLink from "../LearnMoreLink.vue";
import { MiniActionButton } from "../v2";
import MaskingRuleConfig from "./components/MaskingRuleConfig.vue";
import {
  getClassificationLevelOptions,
  getInstanceIdOptions,
} from "./components/utils";

type MaskingRuleMode = "NORMAL" | "EDIT" | "CREATE";

interface MaskingRuleItem {
  mode: MaskingRuleMode;
  rule: MaskingRulePolicy_MaskingRule;
}

interface LocalState {
  maskingRuleItemList: MaskingRuleItem[];
  processing: boolean;
  reorderRules: boolean;
}
const props = defineProps<{
  embedded?: boolean;
}>();
const { t } = useI18n();
const state = reactive<LocalState>({
  maskingRuleItemList: [],
  processing: false,
  reorderRules: false,
});

const policyStore = usePolicyV1Store();
const hasPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});
const hasSensitiveDataFeature = featureToRef(PlanFeature.FEATURE_DATA_MASKING);
const layout = {
  mainContainerRef: ref<HTMLDivElement>(),
};
if (!props.embedded) {
  layout.mainContainerRef = useBodyLayoutContext().mainContainerRef;
}

const updateList = async () => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: "",
    policyType: PolicyType.MASKING_RULE,
  });
  if (!policy) {
    return;
  }

  state.maskingRuleItemList = (
    policy.policy?.case === "maskingRulePolicy" ? policy.policy.value.rules : []
  ).map((rule) => {
    return {
      mode: "NORMAL",
      rule,
    };
  });
};

onMounted(async () => {
  await updateList();
});

const addNewRule = () => {
  state.maskingRuleItemList.push({
    mode: "CREATE",
    rule: create(MaskingRulePolicy_MaskingRuleSchema, {
      id: uuidv4(),
    }),
  });
  nextTick(() => {
    layout.mainContainerRef.value?.scrollTo(
      0,
      document.body.scrollHeight || document.documentElement.scrollHeight
    );
  });
};

const onEdit = (index: number) => {
  state.maskingRuleItemList[index].mode = "EDIT";
};

const onCancel = (index: number) => {
  const item = state.maskingRuleItemList[index];

  if (item.mode === "CREATE") {
    state.maskingRuleItemList.splice(index, 1);
  } else {
    state.maskingRuleItemList[index] = {
      ...item,
      mode: "NORMAL",
    };
  }
};

const onConfirm = async (rule: MaskingRulePolicy_MaskingRule) => {
  if (state.processing) {
    return;
  }

  const index = state.maskingRuleItemList.findIndex(
    (item) => item.rule.id === rule.id
  );
  if (index < 0) {
    return;
  }

  state.processing = true;
  const isCreate = state.maskingRuleItemList[index].mode === "CREATE";

  state.maskingRuleItemList[index] = {
    mode: "NORMAL",
    rule,
  };
  try {
    await onPolicyUpsert();
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t(`common.${isCreate ? "created" : "updated"}`),
    });
  } finally {
    state.processing = false;
  }
};

const onReorder = (maskingRuleItem: MaskingRuleItem, offset: number) => {
  const index = state.maskingRuleItemList.findIndex(
    (item) => item.rule.id === maskingRuleItem.rule.id
  );
  if (index < 0) {
    return;
  }

  arraySwap(state.maskingRuleItemList, index, index + offset);
};

const onReorderSubmit = async () => {
  await onPolicyUpsert();
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  state.reorderRules = false;
};

const onRuleDelete = async (index: number) => {
  const item = state.maskingRuleItemList[index];

  state.maskingRuleItemList.splice(index, 1);
  if (item.mode === "CREATE") {
    return;
  }

  await onPolicyUpsert();
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.deleted"),
  });
};

const onPolicyUpsert = async () => {
  const patch: Partial<Policy> = {
    type: PolicyType.MASKING_RULE,
    resourceType: PolicyResourceType.WORKSPACE,
    policy: {
      case: "maskingRulePolicy",
      value: create(MaskingRulePolicySchema, {
        rules: state.maskingRuleItemList
          .filter((item) => item.mode === "NORMAL")
          .map((item) => item.rule),
      }),
    },
  };

  await policyStore.upsertPolicy({
    parentPath: "",
    policy: patch,
  });
};

const factorList = computed((): Factor[] => {
  const list: Factor[] = [
    "environment_id", // using `environment.resource_id`
    "project_id", // using `project.resource_id`
    "instance_id", // using `instance.resource_id`
    "database_name",
    "table_name",
    "column_name",
    "classification_level",
  ];

  return list;
});

const factorOptionsMap = computed((): Map<Factor, OptionConfig> => {
  return factorList.value.reduce((map, factor) => {
    let options: SelectOption[] = [];
    switch (factor) {
      case "environment_id":
        options = getEnvironmentIdOptions();
        break;
      case "instance_id":
        const store = useInstanceV1Store();
        map.set(factor, {
          remote: true,
          options: [],
          search: async (keyword: string) => {
            return store
              .fetchInstanceList({
                pageSize: getDefaultPagination(),
                filter: {
                  query: keyword,
                },
              })
              .then((resp) => getInstanceIdOptions(resp.instances));
          },
        });
        return map;
      case "project_id":
        const projectStore = useProjectV1Store();
        map.set(factor, {
          remote: true,
          options: [],
          search: async (keyword: string) => {
            return projectStore
              .fetchProjectList({
                pageSize: getDefaultPagination(),
                filter: {
                  query: keyword,
                },
              })
              .then((resp) => getProjectIdOptions(resp.projects));
          },
        });
        return map;
      case "classification_level":
        options = getClassificationLevelOptions();
        break;
    }
    map.set(factor, {
      remote: false,
      options,
    });
    return map;
  }, new Map<Factor, OptionConfig>());
});
</script>
