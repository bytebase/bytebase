<template>
  <div>
    <BBTable
      :column-list="COLUMN_LIST"
      :data-source="tableRows"
      :show-header="false"
      :left-bordered="true"
      :right-bordered="true"
      :top-bordered="true"
      :bottom-bordered="true"
      :row-clickable="false"
    >
      <template #body="{ rowData: row }: { rowData: TableRow }">
        <BBTableCell :left-padding="4" class="w-[1%]">
          <!-- as narrow as possible -->
          <div
            class="relative w-5 h-5 flex flex-shrink-0 items-center justify-center rounded-full select-none"
            :class="statusIconClass(row.checkResult.status)"
          >
            <template v-if="row.checkResult.status == 'SUCCESS'">
              <heroicons-solid:check class="w-4 h-4" />
            </template>
            <template v-if="row.checkResult.status == 'WARN'">
              <heroicons-outline:exclamation class="h-4 w-4" />
            </template>
            <template v-else-if="row.checkResult.status == 'ERROR'">
              <span class="text-white font-medium text-base" aria-hidden="true"
                >!</span
              >
            </template>
          </div>
        </BBTableCell>
        <BBTableCell
          v-if="showCategoryColumn"
          class="min-w-[4rem] max-w-[6rem] whitespace-nowrap"
        >
          {{ row.category }}
        </BBTableCell>
        <BBTableCell class="w-[12rem] break-all">
          {{ row.title }}
        </BBTableCell>
        <BBTableCell class="w-auto">
          <span>{{ row.checkResult.content }}</span>
          <template v-if="row.checkResult.details">
            <span
              class="ml-1 normal-link"
              @click="state.activeResultDefinition = row.checkResult.details"
              >{{ $t("sql-review.view-definition") }}</span
            >
            <span class="border-r border-control-border ml-1"></span>
          </template>
          <template
            v-if="row.checkResult.namespace === 'bb.advisor' && getActiveRule(row.checkResult.title as RuleType)"
          >
            <span
              class="ml-1 normal-link"
              @click="setActiveRule(row.checkResult.title as RuleType)"
              >{{ $t("sql-review.rule-detail") }}</span
            >
            <span class="border-r border-control-border ml-1"></span>
          </template>
          <a
            v-if="row.link"
            class="ml-1 normal-link"
            :href="row.link.url"
            :target="row.link.target"
          >
            {{ row.link.title }}
          </a>
        </BBTableCell>
      </template>
    </BBTable>

    <SQLRuleEditDialog
      v-if="state.activeRule"
      :editable="false"
      :rule="state.activeRule.rule"
      :payload="state.activeRule.payload"
      :disabled="false"
      @cancel="state.activeRule = undefined"
    />

    <TaskCheckResultDefinitionModal
      v-if="state.activeResultDefinition"
      :definition="state.activeResultDefinition"
      @cancel="state.activeResultDefinition = undefined"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  TaskCheckStatus,
  TaskCheckRun,
  TaskCheckResult,
  GeneralErrorCode,
  ruleTemplateMap,
  getRuleLocalization,
  SQLReviewPolicyErrorCode,
  RuleType,
  RuleTemplate,
  Task,
  findRuleTemplate,
} from "@/types";
import type { BBTableColumn } from "@/bbkit";
import { LocalizedSQLRuleErrorCodes } from "./const";
import { SQLRuleEditDialog } from "@/components/SQLReview/components";
import { PayloadValueType } from "../SQLReview/components/RuleConfigComponents";
import { useReviewPolicyByEnvironmentId } from "@/store";

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
  checkResult: TaskCheckResult;
  category: string;
  title: string;
  link: ErrorCodeLink | undefined;
};

type LocalState = {
  activeRule: PreviewSQLReviewRule | undefined;
  activeResultDefinition?: string;
};

const props = defineProps({
  taskCheckRun: {
    required: true,
    type: Object as PropType<TaskCheckRun>,
  },
  task: {
    required: true,
    type: Object as PropType<Task>,
  },
});

const state = reactive<LocalState>({
  activeRule: undefined,
});

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
        line: undefined,
        column: undefined,
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
        line: undefined,
        column: undefined,
      },
    ];
  }

  return [];
});

const categoryAndTitle = (checkResult: TaskCheckResult): [string, string] => {
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
  checkResult: TaskCheckResult
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
      const url = `https://www.bytebase.com/docs/reference/error-code/${
        checkResult.namespace === "bb.advisor" ? "advisor" : "core"
      }?source=console#${checkResult.code}`;
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

const COLUMN_LIST = computed((): BBTableColumn[] => {
  const STATUS = {
    title: "Status",
  };
  const CATEGORY = {
    title: "Category",
  };
  const TITLE = {
    title: "Title",
  };
  const CONTENT = {
    title: "Detail",
  };
  if (showCategoryColumn.value) {
    return [STATUS, CATEGORY, TITLE, CONTENT];
  }
  return [STATUS, TITLE, CONTENT];
});

const reviewPolicy = useReviewPolicyByEnvironmentId(
  computed(() => String(props.task.instance.environment.id))
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
