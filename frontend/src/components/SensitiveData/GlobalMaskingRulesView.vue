<template>
  <div class="w-full mt-4 space-y-4">
    <div class="flex items-center justify-between">
      <!-- Filter -->
      <div class="flex items-center gap-x-3">
        <div class="textlabel">
          {{ $t("settings.sensitive-data.masking-level.self") }}
        </div>
        <label
          v-for="item in levelFilterItemList"
          :key="item.value"
          class="flex items-center gap-x-2 text-sm text-gray-600"
        >
          <NCheckbox
            :checked="state.checkedLevel.has(item.value)"
            @update:checked="toggleCheckLevel(item.value, $event)"
          >
            <BBBadge
              class="whitespace-nowrap"
              :text="item.label"
              :can-remove="false"
              :badge-style="item.style"
              size="small"
            />
          </NCheckbox>
        </label>
      </div>
      <div v-if="state.reorderRules" class="flex items-center space-x-3">
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
      <div v-else class="flex items-center space-x-3">
        <NButton
          secondary
          type="primary"
          :disabled="
            !hasPermission ||
            !hasSensitiveDataFeature ||
            state.maskingRuleItemList.length <= 1
          "
          @click="
            () => {
              state.checkedLevel.clear();
              state.reorderRules = true;
            }
          "
        >
          {{ $t("settings.sensitive-data.global-rules.re-order") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!hasPermission || !hasSensitiveDataFeature"
          @click="addNewRule"
        >
          {{ $t("settings.sensitive-data.global-rules.add-rule") }}
        </NButton>
      </div>
    </div>
    <div class="space-y-5 divide-y-2 pb-10 divide-gray-100">
      <div
        v-if="filteredRuleList.length === 0"
        class="border-4 border-dashed border-gray-200 rounded-lg h-96 flex justify-center items-center"
      >
        <div class="text-center flex flex-col justify-center items-center">
          <img src="../../assets/illustration/no-data.webp" class="w-52" />
        </div>
      </div>
      <div
        v-for="(item, index) in filteredRuleList"
        :key="item.rule.id"
        class="flex items-start space-x-5"
      >
        <div v-if="state.reorderRules" class="mt-4">
          <button @click="onReorder(item, -1)">
            <heroicons-solid:arrow-circle-up
              v-if="index !== 0"
              class="w-6 h-6"
            />
          </button>
          <button @click="onReorder(item, 1)">
            <heroicons-solid:arrow-circle-down
              v-if="index !== filteredRuleList.length - 1"
              class="w-6 h-6"
            />
          </button>
        </div>
        <NPopconfirm v-else @positive-click="onRuleDelete(item)">
          <template #trigger>
            <button
              class="mt-4 w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
              @click.stop=""
            >
              <heroicons-outline:trash class="w-4 h-4" />
            </button>
          </template>
          <div class="whitespace-nowrap">
            {{ $t("settings.sensitive-data.global-rules.delete-rule-tip") }}
          </div>
        </NPopconfirm>
        <MaskingRuleConfig
          :readonly="!hasPermission || state.reorderRules"
          :disabled="state.processing"
          :masking-rule="item.rule"
          :is-create="item.mode === 'CREATE'"
          @cancel="onCancel(item.rule.id, item.mode)"
          @confirm="(rule: MaskingRulePolicy_MaskingRule) => onConfirm(rule)"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NCheckbox, NPopconfirm } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, nextTick, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import BBBadge, { type BBBadgeStyle } from "@/bbkit/BBBadge.vue";
import {
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  usePolicyV1Store,
} from "@/store";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
  MaskingRulePolicy_MaskingRule,
} from "@/types/proto/v1/org_policy_service";
import { arraySwap, hasWorkspacePermissionV1 } from "@/utils";

type MaskingRuleMode = "CREATE" | "EDIT";

type LevelFilterItem = {
  value: number;
  label: string;
  style: BBBadgeStyle;
};

interface MaskingRuleItem {
  mode: MaskingRuleMode;
  rule: MaskingRulePolicy_MaskingRule;
}

interface LocalState {
  checkedLevel: Set<MaskingLevel>;
  maskingRuleItemList: MaskingRuleItem[];
  processing: boolean;
  reorderRules: boolean;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  checkedLevel: new Set<MaskingLevel>(),
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
        mode: "EDIT",
        rule,
      };
    }
  );
};

onMounted(async () => {
  await updateList();
});

const levelFilterItemList = computed(() => {
  return [
    MaskingLevel.FULL,
    MaskingLevel.PARTIAL,
    MaskingLevel.NONE,
  ].map<LevelFilterItem>((level) => {
    return {
      value: level,
      label: t(
        `settings.sensitive-data.masking-level.${maskingLevelToJSON(
          level
        ).toLowerCase()}`
      ),
      style:
        level === MaskingLevel.FULL
          ? "CRITICAL"
          : level === MaskingLevel.PARTIAL
          ? "WARN"
          : level === MaskingLevel.NONE
          ? "INFO"
          : "DISABLED",
    };
  });
});

const filteredRuleList = computed(() => {
  return state.maskingRuleItemList.filter((item) => {
    return (
      state.checkedLevel.size === 0 ||
      state.checkedLevel.has(item.rule.maskingLevel)
    );
  });
});

const toggleCheckLevel = (level: number, checked: boolean) => {
  if (checked) state.checkedLevel.add(level);
  else state.checkedLevel.delete(level);
};

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

const onCancel = (id: string, mode: MaskingRuleMode) => {
  if (mode !== "CREATE") {
    return;
  }
  const index = state.maskingRuleItemList.findIndex(
    (item) => item.rule.id === id
  );
  if (index < 0) {
    return;
  }
  state.maskingRuleItemList.splice(index, 1);
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

  state.maskingRuleItemList[index] = {
    mode: "EDIT",
    rule,
  };
  try {
    await onPolicyUpsert();
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
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

const onRuleDelete = async (maskingRuleItem: MaskingRuleItem) => {
  const index = state.maskingRuleItemList.findIndex(
    (item) => item.rule.id === maskingRuleItem.rule.id
  );
  if (index < 0) {
    return;
  }

  state.maskingRuleItemList.splice(index, 1);
  if (maskingRuleItem.mode === "CREATE") {
    return;
  }

  await onPolicyUpsert();
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const onPolicyUpsert = async () => {
  const patch: Partial<Policy> = {
    type: PolicyType.MASKING_RULE,
    resourceType: PolicyResourceType.WORKSPACE,
    resourceUid: "1",
    maskingRulePolicy: {
      rules: state.maskingRuleItemList
        .filter((item) => item.mode === "EDIT")
        .map((item) => item.rule),
    },
  };

  await policyStore.upsertPolicy({
    parentPath: "",
    policy: patch,
    updateMask: ["payload"],
  });
};
</script>
