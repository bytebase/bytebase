<template>
  <div>
    <BBTable
      :column-list="columnList"
      :data-source="checkResultList"
      :show-header="false"
      :left-bordered="true"
      :right-bordered="true"
      :top-bordered="true"
      :bottom-bordered="true"
      :row-clickable="false"
    >
      <template #body="{ rowData: checkResult }">
        <BBTableCell :left-padding="4" class="w-16">
          <div class="flex gap-x-3">
            <div
              class="relative w-5 h-5 flex flex-shrink-0 items-center justify-center rounded-full select-none"
              :class="statusIconClass(checkResult.status)"
            >
              <template v-if="checkResult.status == 'SUCCESS'">
                <heroicons-solid:check class="w-4 h-4" />
              </template>
              <template v-if="checkResult.status == 'WARN'">
                <heroicons-outline:exclamation class="h-4 w-4" />
              </template>
              <template v-else-if="checkResult.status == 'ERROR'">
                <span
                  class="text-white font-medium text-base"
                  aria-hidden="true"
                  >!</span
                >
              </template>
            </div>
            {{ errorTitle(checkResult) }}
          </div>
        </BBTableCell>
        <BBTableCell class="w-64">
          {{ checkResult.content }}
          <a
            v-if="errorCodeLink(checkResult)"
            class="normal-link"
            :href="errorCodeLink(checkResult)?.url"
            :target="errorCodeLink(checkResult)?.target"
            >{{ errorCodeLink(checkResult)?.title }}</a
          >
        </BBTableCell>
      </template>
    </BBTable>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { useI18n } from "vue-i18n";
import {
  TaskCheckStatus,
  TaskCheckRun,
  TaskCheckResult,
  GeneralErrorCode,
  ruleTemplateMap,
  getRuleLocalization,
  CompatibilityErrorCode,
  SQLReviewPolicyErrorCode,
  RuleType,
} from "@/types";

const columnList = computed(() => [
  {
    title: "",
  },
  {
    title: "Title",
  },
  {
    title: "Detail",
  },
]);

interface ErrorCodeLink {
  title: string;
  target: string;
  url: string;
}

export default defineComponent({
  name: "TaskCheckRunPanel",
  components: {},
  props: {
    taskCheckRun: {
      required: true,
      type: Object as PropType<TaskCheckRun>,
    },
  },
  setup(props) {
    const { t } = useI18n();

    const statusIconClass = (status: TaskCheckStatus) => {
      switch (status) {
        case "SUCCESS":
          return "bg-success text-white";
        case "WARN":
          return "bg-warning text-white";
        case "ERROR":
          return "bg-error text-white";
      }
    };

    const checkResultList = computed((): TaskCheckResult[] => {
      if (props.taskCheckRun.status == "DONE") {
        return props.taskCheckRun.result.resultList;
      } else if (props.taskCheckRun.status == "FAILED") {
        return [
          {
            status: "ERROR",
            title: t("common.error"),
            code: props.taskCheckRun.code,
            content: props.taskCheckRun.result.detail,
            namespace: "bb.core",
          },
        ];
      } else if (props.taskCheckRun.status == "CANCELED") {
        return [
          {
            status: "WARN",
            title: t("common.canceled"),
            code: props.taskCheckRun.code,
            content: "",
            namespace: "bb.core",
          },
        ];
      }

      return [];
    });

    const errorTitle = (checkResult: TaskCheckResult): string => {
      let title = "";

      switch (checkResult.code) {
        case SQLReviewPolicyErrorCode.EMPTY_POLICY:
          title = checkResult.title;
          break;
        // SchemaSQLPolicyErrorCode
        case SQLReviewPolicyErrorCode.STATEMENT_NO_WHERE:
        case SQLReviewPolicyErrorCode.STATEMENT_NO_SELECT_ALL:
        case SQLReviewPolicyErrorCode.STATEMENT_LEADING_WILDCARD_LIKE:
        case SQLReviewPolicyErrorCode.TABLE_NAMING_DISMATCH:
        case SQLReviewPolicyErrorCode.COLUMN_NAMING_DISMATCH:
        case SQLReviewPolicyErrorCode.INDEX_NAMING_DISMATCH:
        case SQLReviewPolicyErrorCode.UK_NAMING_DISMATCH:
        case SQLReviewPolicyErrorCode.FK_NAMING_DISMATCH:
        case SQLReviewPolicyErrorCode.NO_REQUIRED_COLUMN:
        case SQLReviewPolicyErrorCode.COLUMN_CANBE_NULL:
        case SQLReviewPolicyErrorCode.NOT_INNODB_ENGINE:
        case SQLReviewPolicyErrorCode.NO_PK_IN_TABLE:
        case SQLReviewPolicyErrorCode.FK_IN_TABLE:
        case SQLReviewPolicyErrorCode.TABLE_DROP_NAMING_CONVENTION:
        case SQLReviewPolicyErrorCode.DATABASE_NOT_EMPTY:
        case CompatibilityErrorCode.COMPATIBILITY_DROP_DATABASE:
        case CompatibilityErrorCode.COMPATIBILITY_RENAME_TABLE:
        case CompatibilityErrorCode.COMPATIBILITY_DROP_TABLE:
        case CompatibilityErrorCode.COMPATIBILITY_RENAME_COLUMN:
        case CompatibilityErrorCode.COMPATIBILITY_DROP_COLUMN:
        case CompatibilityErrorCode.COMPATIBILITY_ADD_PRIMARY_KEY:
        case CompatibilityErrorCode.COMPATIBILITY_ADD_UNIQUE_KEY:
        case CompatibilityErrorCode.COMPATIBILITY_ADD_FOREIGN_KEY:
        case CompatibilityErrorCode.COMPATIBILITY_ADD_CHECK:
        case CompatibilityErrorCode.COMPATIBILITY_ALTER_CHECK:
        case CompatibilityErrorCode.COMPATIBILITY_ALTER_COLUMN: {
          const rule = ruleTemplateMap.get(checkResult.title as RuleType);

          if (rule) {
            const ruleLocalization = getRuleLocalization(rule.type);
            title = `[${t(
              `sql-review.category.${rule.category.toLowerCase()}`
            )}] ${ruleLocalization.title}`;
          } else {
            title = checkResult.title;
          }
          break;
        }
      }

      return title ? `${title} (${checkResult.code})` : checkResult.title;
    };

    const errorCodeLink = (
      checkResult: TaskCheckResult
    ): ErrorCodeLink | undefined => {
      switch (checkResult.code) {
        case GeneralErrorCode.OK:
          return;
        case SQLReviewPolicyErrorCode.EMPTY_POLICY:
          return {
            title: t("sql-review.configure-policy"),
            target: "_self",
            url: "/setting/sql-review",
          };
        default: {
          const url = `https://www.bytebase.com/docs/reference/error-code/${
            checkResult.namespace === "bb.advisor" ? "advisor" : "core"
          }?source=console`;
          return {
            title: t("common.view-doc"),
            target: "__blank",
            url: url,
          };
        }
      }
    };

    return {
      columnList,
      statusIconClass,
      checkResultList,
      errorCodeLink,
      errorTitle,
    };
  },
});
</script>
