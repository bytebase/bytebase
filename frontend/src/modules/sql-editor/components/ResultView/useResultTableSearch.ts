import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ScopeOption, SearchParams } from "@/components/AdvancedSearch";
import { useAppStore } from "@/stores/app";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { isNullOrUndefined } from "@/utils/util";
import { extractSQLRowValuePlain } from "@/utils/v1/sql";
import type { ResultTableColumn, ResultTableRow } from "./types";

const EMPTY_SEARCH: SearchParams = { query: "", scopes: [] };

export const useResultTableSearch = (
  rows: ResultTableRow[],
  columns: ResultTableColumn[],
  resetKey?: unknown
) => {
  const { t } = useTranslation();
  const [params, setParams] = useState<SearchParams>(EMPTY_SEARCH);
  const [candidateActiveIndex, setCandidateActiveIndex] = useState(-1);
  const [candidateRowIndexes, setCandidateRowIndexes] = useState<number[]>([]);
  const wasInNoResultsRef = useRef(false);

  const scopeOptions: ScopeOption[] = useMemo(() => {
    return [
      {
        id: "row-number",
        title: t("sql-editor.search-scope-row-number-title"),
        description: t("sql-editor.search-scope-row-number-description"),
      },
      ...columns.map((column) => ({
        id: column.id,
        title: column.name,
        description: t("sql-editor.search-scope-column-description", {
          type: column.columnType,
        }),
      })),
    ];
  }, [columns, t]);

  const cellValueMatches = useCallback((cell: RowValue, query: string) => {
    const value = extractSQLRowValuePlain(cell);
    if (isNullOrUndefined(value)) return false;
    return String(value).toLowerCase().includes(query.toLowerCase());
  }, []);

  const getNextCandidateRowIndex = useCallback(
    (from: number, search: SearchParams): number => {
      if (search.scopes.length === 0 && !search.query) return -1;
      for (let index = from; index < rows.length; index++) {
        const row = rows[index];
        const scopeMatches = search.scopes.every((scope) => {
          if (!scope.value) return false;
          if (scope.id === "row-number") {
            return index + 1 === Number.parseInt(scope.value, 10);
          }
          const columnIndex = columns.findIndex(
            (column) => column.name === scope.id
          );
          if (columnIndex < 0) return false;
          return cellValueMatches(row.item.values[columnIndex], scope.value);
        });
        if (!scopeMatches) continue;
        if (
          search.query &&
          !row.item.values.some((cell) => cellValueMatches(cell, search.query))
        ) {
          continue;
        }
        return index;
      }
      return -1;
    },
    [cellValueMatches, columns, rows]
  );

  useEffect(() => {
    const first = getNextCandidateRowIndex(0, params);
    const indexes: number[] = [];
    if (first >= 0) {
      indexes.push(first);
      const second = getNextCandidateRowIndex(first + 1, params);
      if (second >= 0) indexes.push(second);
    }
    setCandidateRowIndexes(indexes);
    setCandidateActiveIndex(0);

    const searchActive =
      params.query.trim().length > 0 || params.scopes.length > 0;
    const hasNoResults = searchActive && indexes.length === 0;
    if (hasNoResults && !wasInNoResultsRef.current) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "INFO",
        title: t("sql-editor.search-no-result"),
      });
    }
    wasInNoResultsRef.current = hasNoResults;
  }, [getNextCandidateRowIndex, params, t]);

  useEffect(() => {
    setParams(EMPTY_SEARCH);
    setCandidateActiveIndex(-1);
    setCandidateRowIndexes([]);
  }, [resetKey]);

  const next = useCallback(() => {
    if (candidateActiveIndex >= candidateRowIndexes.length - 1) return;
    const nextIndex = candidateActiveIndex + 1;
    setCandidateActiveIndex(nextIndex);
    if (nextIndex === candidateRowIndexes.length - 1) {
      const currentRow = candidateRowIndexes[nextIndex];
      const more = getNextCandidateRowIndex(currentRow + 1, params);
      if (more >= 0) {
        setCandidateRowIndexes((current) => [...current, more]);
      }
    }
  }, [
    candidateActiveIndex,
    candidateRowIndexes,
    getNextCandidateRowIndex,
    params,
  ]);

  const previous = useCallback(() => {
    if (candidateActiveIndex <= 0) return;
    setCandidateActiveIndex((current) => current - 1);
  }, [candidateActiveIndex]);

  const clear = useCallback(() => setParams(EMPTY_SEARCH), []);

  return {
    params,
    setParams,
    scopeOptions,
    candidateActiveIndex,
    candidateRowIndexes,
    activeRowIndex: candidateRowIndexes[candidateActiveIndex] ?? -1,
    next,
    previous,
    clear,
  };
};
