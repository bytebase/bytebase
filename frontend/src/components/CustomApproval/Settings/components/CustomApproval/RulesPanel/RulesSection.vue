<template>
  <div class="space-y-2">
    <div class="flex items-center justify-start">
      <div class="font-medium text-base">
        {{ sourceText(source) }}
      </div>
    </div>
    <div>
      <BBGrid
        :column-list="COLUMNS"
        :data-source="rows"
        :row-clickable="false"
        row-key="level"
        class="border"
      >
        <template #item="{ item: row }: { item: Row }">
          <div class="bb-grid-cell">
            {{ levelText(row.level) }}
          </div>
          <div class="bb-grid-cell">
            <RuleSelect
              :value="row.rule"
              :link="true"
              :on-update="(rule) => updateRow(row, rule)"
            />
            <RiskTips :level="row.level" :source="source" :rule="row.rule" />
          </div>
        </template>
      </BBGrid>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, type BBGridColumn } from "@/bbkit";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import {
  DEFAULT_RISK_LEVEL,
  ParsedApprovalRule,
  PresetRiskLevelList,
} from "@/types";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { levelText, sourceText, useRiskFilter } from "../../common";
import { useCustomApprovalContext } from "../context";
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

const COLUMNS = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("custom-approval.risk.self"),
      width: "10rem",
    },
    {
      title: t("custom-approval.approval-flow.self"),
      width: "1fr",
    },
  ];
  return columns;
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
