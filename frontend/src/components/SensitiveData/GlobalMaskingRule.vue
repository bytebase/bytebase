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
      <NButton
        type="primary"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="addNewRule"
      >
        {{ $t("settings.sensitive-data.add-rule") }}
      </NButton>
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
      <MaskingRuleConfig
        v-for="item in filteredRuleList"
        :key="item.rule.id"
        :readonly="!hasPermission"
        :masking-rule="item.rule"
        :is-create="item.mode === 'CREATE'"
        @cancel="onCancel(item.rule.id, item.mode)"
        @confirm="(rule: MaskingRulePolicy_MaskingRule) => onConfirm(rule)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, nextTick, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import BBBadge, { type BBBadgeStyle } from "@/bbkit/BBBadge.vue";
import { featureToRef, pushNotification, useCurrentUserV1 } from "@/store";
import { usePolicyListByResourceTypeAndPolicyType } from "@/store/modules/v1/policy";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import {
  PolicyType,
  MaskingRulePolicy_MaskingRule,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV1 } from "@/utils";

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
}

const { t } = useI18n();
const state = reactive<LocalState>({
  checkedLevel: new Set<MaskingLevel>(),
  maskingRuleItemList: [],
});

const currentUserV1 = useCurrentUserV1();
const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

watchEffect(() => {
  const policyList = usePolicyListByResourceTypeAndPolicyType({
    resourceType: PolicyResourceType.WORKSPACE,
    policyType: PolicyType.MASKING_RULE,
    showDeleted: false,
  });
  if (policyList.value.length !== 1) {
    return;
  }

  const policy = policyList.value[0].maskingRulePolicy;
  state.maskingRuleItemList = (policy?.rules ?? []).map((rule) => {
    return {
      mode: "EDIT",
      rule,
    };
  });
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

const onConfirm = (rule: MaskingRulePolicy_MaskingRule) => {
  const index = state.maskingRuleItemList.findIndex(
    (item) => item.rule.id === rule.id
  );
  if (index < 0) {
    return;
  }
  state.maskingRuleItemList[index] = {
    mode: "EDIT",
    rule,
  };
  // TODO: call backend api
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
