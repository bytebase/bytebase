<template>
  <div class="flex flex-col gap-y-0.5">
    <div v-if="showCancelButton">
      <NButton secondary size="small" @click="cancelPlanCheckRun">
        <template #icon>
          <XIcon class="w-4 h-auto" />
        </template>
        {{ $t("common.cancel") }}
      </NButton>
    </div>

    <div
      v-for="(row, i) in highlightTableRows"
      :key="i"
      :class="[
        'py-2 px-2 flex flex-col gap-y-2',
        row.checkResult.status === Advice_Level.ERROR &&
          'border-error border rounded-sm',
        row.checkResult.status === Advice_Level.WARNING &&
          'border-warning border rounded-sm',
      ]"
    >
      <div class="flex items-center gap-x-3">
        <div
          class="relative w-5 h-5 flex shrink-0 items-center justify-center rounded-full select-none"
          :class="statusIconClass(row.checkResult.status)"
        >
          <template v-if="row.checkResult.status === Advice_Level.SUCCESS">
            <heroicons-solid:check class="w-4 h-4" />
          </template>
          <template v-if="row.checkResult.status === Advice_Level.WARNING">
            <heroicons-outline:exclamation class="h-4 w-4" />
          </template>
          <template v-else-if="row.checkResult.status === Advice_Level.ERROR">
            <span class="text-white font-medium text-base" aria-hidden="true">
              !
            </span>
          </template>
        </div>
        <div v-if="showCategoryColumn" class="shrink-0">
          {{ row.category }}
        </div>
        <div class="font-semibold">{{ row.title }}</div>

        <slot name="row-title-extra" :row="row" />
      </div>

      <div class="textinfolabel flex flex-col gap-y-0.5">
        <div>{{ row.checkResult.content }}</div>

        <OnlineMigrationDetail
          v-if="row.checkResult.title === 'advice.online-migration'"
          :row="row"
        />

        <div
          class="flex items-center justify-start divide-x divide-block-border"
        >
          <div
            v-if="
              row.checkResult.report.case === 'sqlReviewReport' &&
              getActiveRule(row.checkResult.title)
            "
            class="pl-2 first:pl-0"
          >
            <span
              class="normal-link"
              @click="setActiveRule(row.checkResult.title)"
              >{{ $t("sql-review.rule-detail") }}</span
            >
          </div>
          <div
            v-if="row.checkResult.report.case === 'sqlSummaryReport'"
            class="pl-2 first:pl-0"
          >
            <span>
              {{ row.checkResult.report.value.affectedRows }}
            </span>
          </div>

          <div class="pl-2 first:pl-0">
            <a
              v-if="row.link"
              class="normal-link"
              :href="row.link.url"
              :target="row.link.target"
            >
              {{ row.link.title }}
            </a>
          </div>

          <!-- Only show the error line for latest plan check run -->
          <div
            v-if="
              showCodeLocation &&
              row.checkResult.report.case === 'sqlReviewReport' &&
              row.checkResult.report.value.startPosition
            "
            class="pl-2 first:pl-0"
          >
            <span
              class="normal-link"
              @click="
                convertPositionLineToMonacoLine(
                  row.checkResult.report.value.startPosition.line
                )
              "
            >
              Line
              {{
                convertPositionLineToMonacoLine(
                  row.checkResult.report.value.startPosition.line
                )
              }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <div
      v-for="(row, i) in standardTableRows"
      :key="i"
      class="py-3 px-2 first:pt-2 flex flex-col gap-y-2"
    >
      <div class="flex items-center gap-x-3">
        <div
          class="relative w-5 h-5 flex shrink-0 items-center justify-center rounded-full select-none"
          :class="statusIconClass(row.checkResult.status)"
        >
          <template v-if="row.checkResult.status === Advice_Level.SUCCESS">
            <heroicons-solid:check class="w-4 h-4" />
          </template>
          <template v-if="row.checkResult.status === Advice_Level.WARNING">
            <heroicons-outline:exclamation class="h-4 w-4" />
          </template>
          <template v-else-if="row.checkResult.status === Advice_Level.ERROR">
            <span class="text-white font-medium text-base" aria-hidden="true">
              !
            </span>
          </template>
        </div>
        <div v-if="showCategoryColumn" class="shrink-0">
          {{ row.category }}
        </div>
        <div class="font-semibold">{{ row.title }}</div>
      </div>
      <div class="textinfolabel">
        <span>{{ row.checkResult.content }}</span>
        <template
          v-if="
            row.checkResult.report.case === 'sqlReviewReport' &&
            getActiveRule(row.checkResult.title)
          "
        >
          <span
            class="ml-1 normal-link"
            @click="setActiveRule(row.checkResult.title)"
            >{{ $t("sql-review.rule-detail") }}</span
          >
          <span class="border-r border-control-border ml-1"></span>
        </template>
        <template v-if="row.checkResult.report.case === 'sqlSummaryReport'">
          {{ row.checkResult.report.value.affectedRows }}
        </template>

        <a
          v-if="row.link"
          class="ml-1 normal-link"
          :href="row.link.url"
          :target="row.link.target"
        >
          {{ row.link.title }}
        </a>

        <!-- Only show the error line for latest plan check run -->
        <template
          v-if="
            showCodeLocation &&
            row.checkResult.report.case === 'sqlReviewReport' &&
            row.checkResult.report.value.startPosition
          "
        >
          <span class="border-r border-control-border ml-1"></span>
          <span
            class="ml-1 normal-link"
            @click="
              handleClickPlanCheckDetailLine(
                convertPositionLineToMonacoLine(
                  row.checkResult.report.value.startPosition.line
                )
              )
            "
          >
            Line
            {{
              convertPositionLineToMonacoLine(
                row.checkResult.report.value.startPosition.line
              )
            }}
          </span>
        </template>
      </div>
    </div>

    <div
      v-if="showSuccessPlaceholder"
      class="py-3 px-2 first:pt-2 flex flex-col gap-y-2"
    >
      <div class="flex items-center gap-x-3">
        <div
          class="relative w-5 h-5 flex shrink-0 items-center justify-center rounded-full select-none"
          :class="statusIconClass(Advice_Level.SUCCESS)"
        >
          <heroicons-solid:check class="w-4 h-4" />
        </div>
        <div class="font-semibold">OK</div>
      </div>
      <div class="textinfolabel">
        <span>
          {{ $t("sql-review.all-checks-passed") }}
        </span>
      </div>
    </div>
  </div>

  <SQLRuleEditDialog
    v-if="state.activeRule"
    :rule="state.activeRule"
    :disabled="true"
    @cancel="state.activeRule = undefined"
  />
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { XIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { SQLRuleEditDialog } from "@/components/SQLReview/components";
import { planServiceClientConnect } from "@/connect";
import { WORKSPACE_ROUTE_SQL_REVIEW } from "@/router/dashboard/workspaceRoutes";
import { useReviewPolicyForDatabase } from "@/store";
import type { ComposedDatabase, RuleTemplateV2 } from "@/types";
import {
  convertPolicyRuleToRuleTemplate,
  GeneralErrorCode,
  getRuleLocalization,
  ruleTemplateMapV2,
  ruleTypeToString,
  SQLReviewPolicyErrorCode,
} from "@/types";
import type {
  PlanCheckRun,
  PlanCheckRun_Result,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  CancelPlanCheckRunRequestSchema,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  SQLReviewRule_Level,
  SQLReviewRule_Type,
} from "@/types/proto-es/v1/review_config_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { convertPositionLineToMonacoLine } from "@/utils/v1/position";
import { usePlanCheckRunContext } from "./context";
import { OnlineMigrationDetail } from "./detail";

interface ErrorCodeLink {
  title: string;
  target: string;
  url: string;
}

export type PlanCheckDetailTableRow = {
  checkResult: PlanCheckRun_Result;
  category: string;
  title: string;
  link: ErrorCodeLink | undefined;
};

type LocalState = {
  activeRule?: RuleTemplateV2;
};

const props = defineProps<{
  planCheckRun: PlanCheckRun;
  showCodeLocation?: boolean;
  database?: ComposedDatabase;
}>();

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  activeRule: undefined,
});

const statusIconClass = (status: Advice_Level) => {
  switch (status) {
    case Advice_Level.SUCCESS:
      return "bg-success text-white";
    case Advice_Level.WARNING:
      return "bg-warning text-white";
    case Advice_Level.ERROR:
      return "bg-error text-white";
  }
};

const checkResultList = computed((): PlanCheckRun_Result[] => {
  if (props.planCheckRun.status === PlanCheckRun_Status.DONE) {
    return props.planCheckRun.results;
  } else if (props.planCheckRun.status === PlanCheckRun_Status.FAILED) {
    return [
      create(PlanCheckRun_ResultSchema, {
        status: Advice_Level.ERROR,
        title: t("common.error"),
        content: props.planCheckRun.error,
      }),
    ];
  } else if (props.planCheckRun.status === PlanCheckRun_Status.CANCELED) {
    return [
      create(PlanCheckRun_ResultSchema, {
        status: Advice_Level.WARNING,
        title: t("common.canceled"),
        content: props.planCheckRun.error,
      }),
    ];
  }

  return [];
});

const showCancelButton = computed(
  // Only allow canceling plan check run when it's running.
  () => props.planCheckRun.status === PlanCheckRun_Status.RUNNING
);

const getRuleTemplateByType = (type: string) => {
  // Convert string to enum
  const typeKey = type as keyof typeof SQLReviewRule_Type;
  const typeEnum = SQLReviewRule_Type[typeKey];
  if (typeEnum === undefined) {
    return;
  }

  if (props.database) {
    return ruleTemplateMapV2
      .get(props.database.instanceResource.engine)
      ?.get(typeEnum);
  }

  // fallback
  for (const mapByType of ruleTemplateMapV2.values()) {
    if (mapByType.has(typeEnum)) {
      return mapByType.get(typeEnum);
    }
  }
  return;
};

const isBuiltinRule = (type: string) => {
  return type.startsWith("builtin.");
};

const builtinRuleType = (type: string): SQLReviewRule_Type | undefined => {
  // Convert dot-separated to SCREAMING_SNAKE_CASE
  const typeKey = type
    .toUpperCase()
    .replace(/\./g, "_")
    .replace(/-/g, "_") as keyof typeof SQLReviewRule_Type;
  return SQLReviewRule_Type[typeKey];
};

const builtinRuleLevel = (type: string): SQLReviewRule_Level => {
  switch (type) {
    case "builtin.prior-backup-check":
      return SQLReviewRule_Level.ERROR;
    default:
      return SQLReviewRule_Level.ERROR;
  }
};

const categoryAndTitle = (
  checkResult: PlanCheckRun_Result
): [string, string] => {
  if (checkResult.report.case === "sqlReviewReport") {
    const code = checkResult.code;
    if (!code) {
      return ["", checkResult.title];
    }
    if (code === SQLReviewPolicyErrorCode.EMPTY_POLICY) {
      const title = messageWithCode(checkResult.title, code);
      return ["", title];
    }
    const rule = getRuleTemplateByType(checkResult.title);
    if (rule) {
      const ruleLocalization = getRuleLocalization(
        ruleTypeToString(rule.type),
        rule.engine
      );
      const key = `sql-review.category.${rule.category.toLowerCase()}`;
      const category = t(key);
      const title = messageWithCode(ruleLocalization.title, code);
      return [category, title];
    } else if (isBuiltinRule(checkResult.title)) {
      return [
        t("sql-review.category.builtin"),
        messageWithCode(getRuleLocalization(checkResult.title).title, code),
      ];
    }
    return ["", messageWithCode(checkResult.title, code)];
  }
  if (checkResult.report.case === "sqlSummaryReport") {
    if (typeof checkResult.report.value?.affectedRows === "number") {
      return [
        "",
        `${t("task.check-type.affected-rows.self")} (${t("task.check-type.affected-rows.description")})`,
      ];
    }
    return ["", checkResult.title];
  }

  return ["", checkResult.title];
};

const messageWithCode = (message: string, code: number) => {
  return `${message} #${code}`;
};

const errorCodeLink = (
  checkResult: PlanCheckRun_Result
): ErrorCodeLink | undefined => {
  const code = checkResult.code;
  switch (code) {
    case undefined:
      return;
    case GeneralErrorCode.OK:
      return;
    case SQLReviewPolicyErrorCode.EMPTY_POLICY:
      return {
        title: t("sql-review.configure-policy"),
        target: "__blank",
        url: router.resolve({
          name: WORKSPACE_ROUTE_SQL_REVIEW,
        }).fullPath,
      };
    default: {
      const errorCodeNamespace =
        checkResult.report.case === "sqlReviewReport" ? "advisor" : "core";
      const domain = "https://www.bytebase.com";
      const path = `/docs/reference/error-code/${errorCodeNamespace}/`;
      const query = `source=console#${code}`;
      const url = `${domain}${path}?${query}`;
      return {
        title: t("common.view-doc"),
        target: "__blank",
        url: url,
      };
    }
  }
};

const tableRows = computed(() => {
  return checkResultList.value.map<PlanCheckDetailTableRow>((checkResult) => {
    const [category, title] = categoryAndTitle(checkResult);
    const link = errorCodeLink(checkResult);
    return {
      checkResult,
      category,
      title,
      link,
    };
  });
});

const highlightRowFilter = (row: PlanCheckDetailTableRow) => {
  return row.checkResult.title === "advice.online-migration";
};

const highlightTableRows = computed(() => {
  return tableRows.value.filter(highlightRowFilter);
});

const standardTableRows = computed(() => {
  return tableRows.value.filter((row) => !highlightRowFilter(row));
});

const showSuccessPlaceholder = computed(() => {
  return (
    props.planCheckRun.status === PlanCheckRun_Status.DONE &&
    highlightTableRows.value.length === 0 &&
    standardTableRows.value.length === 0
  );
});

const showCategoryColumn = computed((): boolean =>
  tableRows.value.some((row) => row.category !== "")
);

const reviewPolicy = useReviewPolicyForDatabase(
  computed(() => {
    return props.database;
  })
);

const getActiveRule = (type: string): RuleTemplateV2 | undefined => {
  const engine = props.database?.instanceResource.engine;
  if (isBuiltinRule(type) && engine) {
    const typeEnum = builtinRuleType(type);
    if (!typeEnum) {
      return undefined;
    }
    return {
      type: typeEnum,
      category: "BUILTIN",
      engine: engine,
      level: builtinRuleLevel(type),
      componentList: [],
    };
  }
  // Convert string to enum for comparison
  const typeKey = type as keyof typeof SQLReviewRule_Type;
  const typeEnum = SQLReviewRule_Type[typeKey];

  const rule = reviewPolicy.value?.ruleList.find((rule) => {
    if (engine && rule.engine !== engine) {
      return false;
    }
    return rule.type === typeEnum;
  });
  if (!rule) {
    return undefined;
  }

  const ruleTemplate = getRuleTemplateByType(type);
  if (!ruleTemplate) {
    return undefined;
  }

  return convertPolicyRuleToRuleTemplate(rule, ruleTemplate);
};

const setActiveRule = (type: string) => {
  state.activeRule = getActiveRule(type);
};

const handleClickPlanCheckDetailLine = (line: number) => {
  window.location.hash = `L${line}`;
};

const cancelPlanCheckRun = async () => {
  const request = create(CancelPlanCheckRunRequestSchema, {
    name: props.planCheckRun.name,
  });
  await planServiceClientConnect.cancelPlanCheckRun(request);
  if (usePlanCheckRunContext()) {
    usePlanCheckRunContext().events.emit("status-changed");
  }
};
</script>
