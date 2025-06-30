<template>
  <div class="space-y-2">
    <div class="flex items-center justify-start">
      <div class="font-medium text-base">
        {{ sourceText(source) }}
      </div>
    </div>
    <div>
      <NDataTable
        size="small"
        :columns="columns"
        :data="rows"
        :striped="true"
        :bordered="true"
        :row-key="(row: Row) => String(row.level)"
      />
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import type { ParsedApprovalRule } from "@/types";
import { DEFAULT_RISK_LEVEL, PresetRiskLevelList } from "@/types";
import type { Risk_Source } from "@/types/proto-es/v1/risk_service_pb";
import { levelText, sourceText, useRiskFilter } from "../../common";
import { useCustomApprovalContext } from "../context";
import RiskTips from "./RiskTips.vue";
import RuleSelect from "./RuleSelect.vue";

type Row = {
  level: number;
  rule: string | undefined; // LocalApprovalRule.uid
};

const props = defineProps<{
  source: Risk_Source;
}>();

const { t } = useI18n();
const store = useWorkspaceApprovalSettingStore();
const context = useCustomApprovalContext();

const columns = computed((): DataTableColumn<Row>[] => {
  return [
    {
      title: t("custom-approval.risk.self"),
      key: "level",
      width: 160,
      render: (row) => levelText(row.level),
    },
    {
      title: t("custom-approval.approval-flow.self"),
      key: "rule",
      render: (row) => (
        <div class="flex items-center space-x-2">
          <RuleSelect
            class="flex-1 max-w-md min-w-[10rem]"
            value={row.rule}
            link={true}
            onUpdate={(rule: string | undefined) => updateRow(row, rule)}
          />
          <RiskTips level={row.level} source={props.source} rule={row.rule} />
        </div>
      ),
    },
  ];
});

const filter = useRiskFilter();

const rulesMap = computed(() => {
  const map = new Map<number, ParsedApprovalRule>();
  store.config.parsed
    .filter((item) => item.source === props.source)
    .forEach((item) => {
      map.set(item.level, item);
    });
  return map;
});

const rows = computed(() => {
  const filteredLevelList = [...filter.levels.value.values()];
  filteredLevelList.sort((a, b) => -(a - b)); // by level DESC
  const displayLevelList =
    filteredLevelList.length === 0
      ? [...PresetRiskLevelList.map((item) => item.level), DEFAULT_RISK_LEVEL]
      : filteredLevelList;

  return displayLevelList.map<Row>((level) => ({
    level,
    rule: rulesMap.value.get(level)?.rule ?? "",
  }));
});

const updateRow = async (row: Row, rule: string | undefined) => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }

  const { source } = props;
  const { level } = row;
  try {
    await store.updateRuleFlow(source, level, rule);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } catch {
    // nothing, exception has been handled already
  }
};
</script>
