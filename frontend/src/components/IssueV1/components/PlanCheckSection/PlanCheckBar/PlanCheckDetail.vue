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
        <span>{{ row.checkResult.content }}</span>
        <template v-if="row.checkResult.sqlReviewReport?.detail">
          <span
            class="ml-1 normal-link"
            @click="selectDetail(row.checkResult.sqlReviewReport)"
            >{{ $t("sql-review.view-definition") }}</span
          >
          <span class="border-r border-control-border ml-1"></span>
        </template>
        TODO: details
      </div>
    </template>
  </BBGrid>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGridColumn, BBGridRow, BBGrid } from "@/bbkit";
import { LocalizedSQLRuleErrorCodes } from "@/components/Issue/const";
import {
  GeneralErrorCode,
  RuleType,
  SQLReviewPolicyErrorCode,
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

interface ErrorCodeLink {
  title: string;
  target: string;
  url: string;
}

// type PreviewSQLReviewRule = {
//   rule: RuleTemplate;
//   payload: PayloadValueType[];
// };

type TableRow = {
  checkResult: PlanCheckRun_Result;
  category: string;
  title: string;
  link: ErrorCodeLink | undefined;
};

const props = defineProps<{
  planCheckRun: PlanCheckRun;
  task: Task;
}>();

const { t } = useI18n();

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
    // return [
    //   {
    //     status: "ERROR",
    //     title: t("common.error"),
    //     code: props.planCheckRun.code,
    //     content: props.planCheckRun.result.detail,
    //     namespace: "bb.core",
    //     line: undefined,
    //     column: undefined,
    //   },
    // ];
    return [
      PlanCheckRun_Result.fromJSON({
        status: PlanCheckRun_Result_Status.ERROR,
        title: t("common.error"),
      }),
    ];
  } else if (props.planCheckRun.status === PlanCheckRun_Status.CANCELED) {
    // return [
    //   {
    //     status: "WARN",
    //     title: t("common.canceled"),
    //     code: props.planCheckRun.code,
    //     content: "",
    //     namespace: "bb.core",
    //     line: undefined,
    //     column: undefined,
    //   },
    // ];
    return [
      PlanCheckRun_Result.fromJSON({
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
  if (checkResult.code === SQLReviewPolicyErrorCode.EMPTY_POLICY) {
    const title = messageWithCode(checkResult.title, checkResult.code);
    return ["", title];
  }
  if (LocalizedSQLRuleErrorCodes.has(checkResult.code)) {
    const rule = ruleTemplateMap.get(checkResult.title as RuleType);
    if (rule) {
      const ruleLocalization = getRuleLocalization(rule.type);
      const key = `sql-review.category.${rule.category.toLowerCase()}`;
      const category = t(key);
      const title = messageWithCode(ruleLocalization.title, checkResult.code);
      return [category, title];
    } else {
      return ["", messageWithCode(checkResult.title, checkResult.code)];
    }
  }

  return ["", checkResult.title];
};

const messageWithCode = (message: string, code: number) => {
  return `${message} #${code}`;
};

const errorCodeLink = (
  checkResult: PlanCheckRun_Result
): ErrorCodeLink | undefined => {
  switch (checkResult.code) {
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
      // const url = `https://www.bytebase.com/docs/reference/error-code/${
      //   checkResult.namespace === "bb.advisor" ? "advisor" : "core"
      // }?source=console#${checkResult.code}`;
      const url = "#TODO";
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
    width: "12rem",
  };
  const CONTENT: BBGridColumn = {
    title: "Detail",
    width: "1fr",
  };
  if (showCategoryColumn.value) {
    return [STATUS, CATEGORY, TITLE, CONTENT];
  }
  return [STATUS, TITLE, CONTENT];
});

const selectDetail = (args: any) => {
  // TODO
};
</script>
