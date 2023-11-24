<template>
  <div class="w-full space-y-4">
    <div class="flex flex-row items-center justify-end">
      <div v-if="state.reorderRules" class="flex items-center gap-x-3">
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
      <div v-else class="flex items-center gap-x-3">
        <NButton
          secondary
          type="primary"
          :disabled="
            !hasPermission ||
            !hasSensitiveDataFeature ||
            state.maskingRuleItemList.length <= 1
          "
          @click="state.reorderRules = true"
        >
          {{ $t("settings.sensitive-data.global-rules.re-order") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!hasPermission || !hasSensitiveDataFeature"
          @click="addNewRule"
        >
          {{ $t("common.add") }}
        </NButton>
      </div>
    </div>
    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.global-rules.description") }}
      <LearnMoreLink
        url="https://www.bytebase.com/docs/security/mask-data?source=console"
      />
    </div>
    <NoDataPlaceholder v-if="state.maskingRuleItemList.length === 0" />
    <div
      v-for="(item, index) in state.maskingRuleItemList"
      :key="item.rule.id"
      class="flex items-start gap-x-5 pt-4"
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
      <div :class="['w-full', item.mode === 'NORMAL' ? '' : 'ml-[42px]']">
        <MaskingRuleConfig
          :key="`expr-${item.rule.id}`"
          :index="index + 1"
          :disabled="state.processing"
          :masking-rule="item.rule"
          :readonly="item.mode === 'NORMAL' || state.reorderRules"
          :factor-list="factorList"
          :factor-options-map="factorOptionsMap"
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
import { PencilIcon } from "lucide-vue-next";
import { ChevronUpIcon } from "lucide-vue-next";
import { ChevronDownIcon } from "lucide-vue-next";
import type { SelectOption } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, nextTick, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { Factor } from "@/plugins/cel";
import {
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  usePolicyV1Store,
} from "@/store";
import { MaskingLevel } from "@/types/proto/v1/common";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
  MaskingRulePolicy_MaskingRule,
} from "@/types/proto/v1/org_policy_service";
import { arraySwap, hasWorkspacePermissionV1 } from "@/utils";
import NoData from "../misc/NoData.vue";
import { MiniActionButton } from "../v2";
import MaskingRuleConfig from "./components/MaskingRuleConfig.vue";
import {
  getClassificationLevelOptions,
  getEnvironmentIdOptions,
  getInstanceIdOptions,
  getProjectIdOptions,
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

const { t } = useI18n();
const state = reactive<LocalState>({
  maskingRuleItemList: [],
  processing: false,
  reorderRules: false,
});

const policyStore = usePolicyV1Store();
const currentUserV1 = useCurrentUserV1();
const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const updateList = async () => {
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: "",
    policyType: PolicyType.MASKING_RULE,
  });
  if (!policy) {
    return;
  }

  state.maskingRuleItemList = (policy.maskingRulePolicy?.rules ?? []).map(
    (rule) => {
      return {
        mode: "NORMAL",
        rule,
      };
    }
  );
};

onMounted(async () => {
  await updateList();
});

const addNewRule = () => {
  state.maskingRuleItemList.push({
    mode: "CREATE",
    rule: MaskingRulePolicy_MaskingRule.fromJSON({
      id: uuidv4(),
      maskingLevel: MaskingLevel.FULL,
    }),
  });
  nextTick(() => {
    const elem = document.querySelector("#bb-layout-main");
    elem?.scrollTo(
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
    resourceUid: "1",
    maskingRulePolicy: {
      rules: state.maskingRuleItemList
        .filter((item) => item.mode === "NORMAL")
        .map((item) => item.rule),
    },
  };

  await policyStore.upsertPolicy({
    parentPath: "",
    policy: patch,
    updateMask: ["payload"],
  });
};

const factorList = computed((): Factor[] => {
  const list = [
    "environment_id", // using `environment.resource_id`
    "project_id", // using `project.resource_id`
    "instance_id", // using `instance.resource_id`
    "database_name",
    "table_name",
    "column_name",
  ];

  const classificationOptions = getClassificationLevelOptions();
  if (classificationOptions.length > 0) {
    list.push("classification_level");
  }

  return list;
});

const factorOptionsMap = computed((): Map<Factor, SelectOption[]> => {
  return factorList.value.reduce((map, factor) => {
    let options: SelectOption[] = [];
    switch (factor) {
      case "environment_id":
        options = getEnvironmentIdOptions();
        break;
      case "instance_id":
        options = getInstanceIdOptions();
        break;
      case "project_id":
        options = getProjectIdOptions();
        break;
      case "classification_level":
        options = getClassificationLevelOptions();
        break;
    }
    map.set(factor, options);
    return map;
  }, new Map<Factor, SelectOption[]>());
});
</script>
