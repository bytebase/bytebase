import { type ComputedRef, computed, type Ref, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useSheetV1Store } from "@/store";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { getSheetStatement, sheetNameOfTaskV1 } from "@/utils";
import {
  MAX_STATEMENT_DISPLAY_SIZE,
  STATEMENT_PREVIEW_LENGTH,
} from "../constants";

export interface UseTaskStatementReturn {
  loading: Ref<boolean>;
  statement: Ref<string>;
  statementPreview: ComputedRef<string>;
  displayedStatement: ComputedRef<string>;
  isStatementTruncated: ComputedRef<boolean>;
}

export const useTaskStatement = (
  task: () => Task,
  isExpanded: () => boolean
): UseTaskStatementReturn => {
  const { t } = useI18n();
  const sheetStore = useSheetV1Store();

  const loading = ref(false);
  const statement = ref("");

  // Load statement - uses cached version for collapsed view, lazy loads for expanded view
  watchEffect(async () => {
    const sheetName = sheetNameOfTaskV1(task());
    if (!sheetName) {
      statement.value = "";
      return;
    }

    // Check cache first for both expanded and collapsed views
    const cachedSheet = sheetStore.getSheetByName(sheetName);

    if (cachedSheet) {
      // Use cached statement immediately
      statement.value = getSheetStatement(cachedSheet);
    } else if (isExpanded()) {
      // Only fetch if expanded and not in cache
      loading.value = true;
      try {
        await sheetStore.getOrFetchSheetByName(sheetName);
        const sheet = sheetStore.getSheetByName(sheetName);
        statement.value = sheet ? getSheetStatement(sheet) : "";
      } finally {
        loading.value = false;
      }
    } else {
      // Collapsed and not in cache: show empty (will be loaded when expanded)
      statement.value = "";
    }
  });

  // Preview for collapsed view (first line, truncated)
  const statementPreview = computed(() => {
    const stmt = statement.value;
    if (!stmt) {
      return t("rollout.task.no-statement");
    }
    const firstLine = stmt.split("\n")[0];
    return firstLine.length > STATEMENT_PREVIEW_LENGTH
      ? firstLine.substring(0, STATEMENT_PREVIEW_LENGTH) + "..."
      : firstLine;
  });

  // Check if statement exceeds display limit
  const isStatementTruncated = computed(() => {
    return statement.value.length > MAX_STATEMENT_DISPLAY_SIZE;
  });

  // Statement for expanded view (truncated if too large for performance)
  const displayedStatement = computed(() => {
    if (!statement.value) {
      return t("rollout.task.no-statement");
    }
    if (isStatementTruncated.value) {
      return statement.value.substring(0, MAX_STATEMENT_DISPLAY_SIZE);
    }
    return statement.value;
  });

  return {
    loading,
    statement,
    statementPreview,
    displayedStatement,
    isStatementTruncated,
  };
};
