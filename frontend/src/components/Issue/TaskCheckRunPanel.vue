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
        <BBTableCell :left-padding="4" class="table-cell w-4">
          <div class="flex flex-row space-x-2">
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
          </div>
        </BBTableCell>
        <BBTableCell class="table-cell w-16">
          {{ checkResult.title }}
        </BBTableCell>
        <BBTableCell class="table-cell w-48">
          {{ checkResult.content }}
          <a
            v-if="errorCodeLink(checkResult.code)"
            class="normal-link"
            :href="errorCodeLink(checkResult.code)"
            target="__blank"
            >view doc</a
          >
        </BBTableCell>
      </template>
    </BBTable>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { BBTableColumn } from "../../bbkit/types";
import {
  TaskCheckStatus,
  TaskCheckRun,
  TaskCheckResult,
  ErrorCode,
  ERROR_LIST,
} from "../../types";

const columnList: BBTableColumn[] = [
  {
    title: "",
  },
  {
    title: "Title",
  },
  {
    title: "Detail",
  },
];

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
          },
        ];
      } else if (props.taskCheckRun.status == "CANCELED") {
        return [
          {
            status: "WARN",
            title: t("common.canceled"),
            code: props.taskCheckRun.code,
            content: "",
          },
        ];
      }

      return [];
    });

    const errorCodeLink = (code: ErrorCode): string => {
      const error = ERROR_LIST.find((item) => item.code == code);
      return error ? `https://bytebase.com/docs/error-code#${error.hash}` : "";
    };

    return {
      columnList,
      statusIconClass,
      checkResultList,
      errorCodeLink,
    };
  },
});
</script>
