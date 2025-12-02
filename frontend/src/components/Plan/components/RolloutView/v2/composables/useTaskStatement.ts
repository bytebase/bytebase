import { type ComputedRef, computed, type Ref, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useSheetV1Store } from "@/store";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { getSheetStatement, sheetNameOfTaskV1 } from "@/utils";
import { MAX_STATEMENT_DISPLAY_SIZE } from "../constants";

export interface UseTaskStatementReturn {
  loading: Ref<boolean>;
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

  // Only fetch statement when expanded (no preview needed for collapsed view)
  watchEffect(async () => {
    if (!isExpanded()) {
      // Skip fetching for collapsed tasks
      return;
    }

    const sheetName = sheetNameOfTaskV1(task());
    if (!sheetName) {
      statement.value = "";
      return;
    }

    // Check cache first
    const cachedSheet = sheetStore.getSheetByName(sheetName);
    if (cachedSheet) {
      statement.value = getSheetStatement(cachedSheet);
      return;
    }

    // Fetch if not in cache
    loading.value = true;
    try {
      await sheetStore.getOrFetchSheetByName(sheetName);
      const sheet = sheetStore.getSheetByName(sheetName);
      statement.value = sheet ? getSheetStatement(sheet) : "";
    } finally {
      loading.value = false;
    }
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
    displayedStatement,
    isStatementTruncated,
  };
};
