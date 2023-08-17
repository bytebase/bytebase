<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="tableRows"
    :show-header="false"
    :row-clickable="false"
    class="border"
  >
    <template #item="{ item: row }: BBGridRow<TableRow>">
      <div class="bb-grid-cell">
        <div
          class="relative w-5 h-5 flex flex-shrink-0 items-center justify-center rounded-full select-none"
          :class="statusIconClass(row.checkResult.status)"
        >
          <template
            v-if="row.checkResult.status === PlanCheckRun_Result_Status.SUCCESS"
          >
            <heroicons-solid:check class="w-4 h-4" />
          </template>
          <template
            v-if="row.checkResult.status === PlanCheckRun_Result_Status.WARNING"
          >
            <heroicons-outline:exclamation class="h-4 w-4" />
          </template>
          <template
            v-else-if="
              row.checkResult.status === PlanCheckRun_Result_Status.ERROR
            "
          >
            <span class="text-white font-medium text-base" aria-hidden="true"
              >!</span
            >
          </template>
        </div>
      </div>
      <div v-if="showCategoryColumn" class="bb-grid-cell">
        {{ row.category }}
      </div>
      <div class="bb-grid-cell">
        {{ row.title }}
      </div>
      <div class="bb-grid-cell">
        <div>
          <span>{{ row.checkResult.content }}</span>
          <template v-if="row.checkResult.sqlReviewReport?.detail">
            <span
              class="ml-1 normal-link"
              @click="
                state.activeResultDefinition =
                  row.checkResult.sqlReviewReport.detail
              "
              >{{ $t("sql-review.view-definition") }}</span
            >
            <span class="border-r border-control-border ml-1"></span>
          </template>
          <template
            v-if="row.checkResult.sqlReviewReport && getActiveRule(row.checkResult.title as RuleType)"
          >
            <span
              class="ml-1 normal-link"
              @click="setActiveRule(row.checkResult.title as RuleType)"
              >{{ $t("sql-review.rule-detail") }}</span
            >
            <span class="border-r border-control-border ml-1"></span>
          </template>
          <template v-if="row.checkResult.sqlSummaryReport">
            {{ row.checkResult.sqlSummaryReport.affectedRows }}
          </template>

          <a
            v-if="row.link"
            class="ml-1 normal-link"
            :href="row.link.url"
            :target="row.link.target"
          >
            {{ row.link.title }}
          </a>
        </div>
      </div>
    </template>
  </BBGrid>

  <SQLRuleEditDialog
    v-if="state.activeRule"
    :editable="false"
    :rule="state.activeRule.rule"
    :payload="state.activeRule.payload"
    :disabled="false"
    @cancel="state.activeRule = undefined"
  />

  <PlanCheckResultDefinitionModal
    v-if="state.activeResultDefinition"
    :definition="state.activeResultDefinition"
    @cancel="state.activeResultDefinition = undefined"
  />
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn, BBGridRow, BBGrid } from "@/bbkit";
import { LocalizedSQLRuleErrorCodes } from "@/components/Issue/const";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import { SQLRuleEditDialog } from "@/components/SQLReview/components";
import { PayloadValueType } from "@/components/SQLReview/components/RuleConfigComponents";
import { useReviewPolicyByEnvironmentId } from "@/store";
import {
  GeneralErrorCode,
  RuleTemplate,
  RuleType,
  SQLReviewPolicyErrorCode,
  findRuleTemplate,
  getRuleLocalization,
  ruleTemplateMap,
} from "@/types";
import {
  PlanCheckRun,
  PlanCheckRun_Result,
  PlanCheckRun_Result_Status,
  PlanCheckRun_Status,
  Task,
} from "@/types/proto/v1/rollout_service";
import PlanCheckResultDefinitionModal from "./PlanCheckResultDefinitionModal.vue";

interface ErrorCodeLink {
  title: string;
  target: string;
  url: string;
}

type PreviewSQLReviewRule = {
  rule: RuleTemplate;
  payload: PayloadValueType[];
};

type TableRow = {
  checkResult: PlanCheckRun_Result;
  category: string;
  title: string;
  link: ErrorCodeLink | undefined;
};

type LocalState = {
  activeRule?: PreviewSQLReviewRule;
  activeResultDefinition?: string;
};

const props = defineProps<{
  planCheckRun: PlanCheckRun;
  task: Task;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  activeRule: undefined,
  activeResultDefinition: undefined,
});
const { issue } = useIssueContext();

const statusIconClass = (status: PlanCheckRun_Result_Status) => {
  switch (status) {
    case PlanCheckRun_Result_Status.SUCCESS:
      return "bg-success text-white";
    case PlanCheckRun_Result_Status.WARNING:
      return "bg-warning text-white";
    case PlanCheckRun_Result_Status.ERROR:
      return "bg-error text-white";
  }
};

const checkResultList = computed((): PlanCheckRun_Result[] => {
  if (props.planCheckRun.status === PlanCheckRun_Status.DONE) {
    return props.planCheckRun.results;
  } else if (props.planCheckRun.status === PlanCheckRun_Status.FAILED) {
    return [
      PlanCheckRun_Result.fromPartial({
        status: PlanCheckRun_Result_Status.ERROR,
        title: t("common.error"),
      }),
    ];
  } else if (props.planCheckRun.status === PlanCheckRun_Status.CANCELED) {
    return [
      PlanCheckRun_Result.fromPartial({
        status: PlanCheckRun_Result_Status.WARNING,
        title: t("common.canceled"),
      }),
    ];
  }

  return [];
});

const categoryAndTitle = (
  checkResult: PlanCheckRun_Result
): [string, string] => {
  if (checkResult.sqlReviewReport) {
    const code = checkResult.sqlReviewReport?.code ?? checkResult.code;
    if (code === SQLReviewPolicyErrorCode.EMPTY_POLICY) {
      const title = messageWithCode(checkResult.title, code);
      return ["", title];
    }
    if (LocalizedSQLRuleErrorCodes.has(code)) {
      const rule = ruleTemplateMap.get(checkResult.title as RuleType);
      if (rule) {
        const ruleLocalization = getRuleLocalization(rule.type);
        const key = `sql-review.category.${rule.category.toLowerCase()}`;
        const category = t(key);
        const title = messageWithCode(ruleLocalization.title, code);
        return [category, title];
      } else {
        return ["", messageWithCode(checkResult.title, code)];
      }
    }
    return ["", checkResult.title];
  }
  if (checkResult.sqlSummaryReport) {
    if (typeof checkResult.sqlSummaryReport.affectedRows === "number") {
      return ["", t("task.check-type.affected-rows")];
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
  const code = checkResult.sqlReviewReport?.code ?? checkResult.code;
  switch (code) {
    case undefined:
      return;
    case GeneralErrorCode.OK:
      return;
    case SQLReviewPolicyErrorCode.EMPTY_POLICY:
      return {
        title: t("sql-review.configure-policy"),
        target: "_self",
        url: "/setting/sql-review",
      };
    default: {
      const errorCodeNamespace =
        checkResult.sqlReviewReport !== undefined ? "advisor" : "core";
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
  return checkResultList.value.map<TableRow>((checkResult) => {
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

const showCategoryColumn = computed((): boolean =>
  tableRows.value.some((row) => row.category !== "")
);

const COLUMN_LIST = computed(() => {
  const STATUS: BBGridColumn = {
    title: "Status",
    width: "auto",
  };
  const CATEGORY: BBGridColumn = {
    title: "Category",
    width: "minmax(4rem, 6rem)",
  };
  const TITLE: BBGridColumn = {
    title: "Title",
    width: "minmax(12rem, 1fr)",
  };
  const CONTENT: BBGridColumn = {
    title: "Detail",
    width: "2fr",
  };
  if (showCategoryColumn.value) {
    return [STATUS, CATEGORY, TITLE, CONTENT];
  }
  return [STATUS, TITLE, CONTENT];
});

const reviewPolicy = useReviewPolicyByEnvironmentId(
  computed(() => {
    const database = databaseForTask(issue.value, props.task);
    return database.effectiveEnvironmentEntity.uid;
  })
);
const getActiveRule = (type: RuleType): PreviewSQLReviewRule | undefined => {
  const rule = reviewPolicy.value?.ruleList.find((rule) => rule.type === type);
  if (!rule) {
    return undefined;
  }

  const ruleTemplate = findRuleTemplate(type);
  if (!ruleTemplate) {
    return undefined;
  }
  ruleTemplate.comment = rule.comment;
  const { componentList } = ruleTemplate;
  const payload = componentList.reduce<PayloadValueType[]>(
    (list, component) => {
      list.push(component.payload.value ?? component.payload.default);
      return list;
    },
    []
  );

  return {
    rule: ruleTemplate,
    payload: payload,
  };
};
const setActiveRule = (type: RuleType) => {
  state.activeRule = getActiveRule(type);
};
</script>
