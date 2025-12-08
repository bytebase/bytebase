<template>
  <div class="flex items-start gap-3 px-3 py-2 border rounded-lg bg-gray-50">
    <component :is="statusIcon" class="w-5 h-5 shrink-0" :class="statusColor" />

    <div class="flex-1 min-w-0 flex flex-col gap-y-1">
      <div class="text-sm font-medium text-main">
        {{ displayTitle }}
      </div>
      <div v-if="content" class="text-sm text-control">
        {{ content }}
      </div>
      <div
        v-if="position && (position.line > 0 || position.column > 0)"
        class="text-sm mt-1 text-control-light"
      >
        <template v-if="position.line > 0">
          {{ $t("common.line") }} {{ position.line }}
        </template>
        <template v-if="position.line > 0 && position.column > 0">, </template>
        <template v-if="position.column > 0">
          {{ $t("common.column") }} {{ position.column }}
        </template>
      </div>
      <div
        v-if="affectedRows !== undefined"
        class="text-sm mt-1 flex items-center gap-1"
      >
        <NTag size="small" round>
          {{ $t("task.check-type.affected-rows.self") }}
        </NTag>
        <span>{{ affectedRows }}</span>
        <span class="text-control opacity-80"
          >({{ $t("task.check-type.affected-rows.description") }})</span
        >
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { AlertCircleIcon, CheckCircleIcon, XCircleIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed } from "vue";
import {
  getRuleLocalization,
  ruleTemplateMapV2,
  ruleTypeToString,
} from "@/types";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";

export interface CheckResultPosition {
  line: number;
  column: number;
}

const props = defineProps<{
  status: "SUCCESS" | "WARNING" | "ERROR";
  title: string;
  content?: string;
  position?: CheckResultPosition;
  affectedRows?: bigint;
  code?: number;
  reportType?: "sqlReviewReport" | "sqlSummaryReport";
}>();

const statusIcon = computed(() => {
  switch (props.status) {
    case "ERROR":
      return XCircleIcon;
    case "WARNING":
      return AlertCircleIcon;
    case "SUCCESS":
      return CheckCircleIcon;
    default:
      return CheckCircleIcon;
  }
});

const statusColor = computed(() => {
  switch (props.status) {
    case "ERROR":
      return "text-error";
    case "WARNING":
      return "text-warning";
    case "SUCCESS":
      return "text-success";
    default:
      return "text-control";
  }
});

const getRuleTemplateByType = (type: string) => {
  // Convert string to enum
  const typeKey = type as keyof typeof SQLReviewRule_Type;
  const typeEnum = SQLReviewRule_Type[typeKey];
  if (typeEnum === undefined) {
    return;
  }

  for (const mapByType of ruleTemplateMapV2.values()) {
    if (mapByType.has(typeEnum)) {
      return mapByType.get(typeEnum);
    }
  }
  return;
};

const messageWithCode = (message: string, code: number | undefined): string => {
  if (code !== undefined && code !== 0) {
    return `${message} #${code}`;
  }
  return message;
};

const displayTitle = computed(() => {
  // Skip localization for certain titles
  if (props.title === "OK" || props.title === "Syntax error") {
    return messageWithCode(props.title, props.code);
  }

  // Only apply SQL review localization if this is a SQL review report
  if (props.reportType === "sqlReviewReport") {
    const code = props.code;
    if (!code) {
      return props.title;
    }

    const rule = getRuleTemplateByType(props.title);
    if (rule) {
      const ruleLocalization = getRuleLocalization(
        ruleTypeToString(rule.type),
        rule.engine
      );
      const title = messageWithCode(ruleLocalization.title, code);
      return title;
    } else if (props.title.startsWith("builtin.")) {
      return messageWithCode(getRuleLocalization(props.title).title, code);
    }
  }

  // Add error code if present
  return messageWithCode(props.title, props.code);
});
</script>
