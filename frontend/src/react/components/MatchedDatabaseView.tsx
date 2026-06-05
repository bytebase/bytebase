import { ChevronRight } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { validateSimpleExpr } from "@/plugins/cel";
import { DatabaseTargetDisplay } from "@/react/components/DatabaseTargetDisplay";
import { useAppStore } from "@/react/stores/app";
import { DEBOUNCE_SEARCH_DELAY, isValidDatabaseName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getDefaultPagination } from "@/utils";

interface MatchedDatabaseViewProps {
  project: string;
  expr: ConditionGroupExpr;
  matchedDatabaseNames?: string[];
}

interface SectionState {
  databases: Database[];
  error?: string;
  loading: boolean;
}

export function MatchedDatabaseView({
  project,
  expr,
  matchedDatabaseNames: presetNames,
}: MatchedDatabaseViewProps) {
  const { t } = useTranslation();

  const [loading, setLoading] = useState(false);
  const [matchingError, setMatchingError] = useState<string>();
  const [matchedNames, setMatchedNames] = useState<string[]>([]);
  const [matchedDbs, setMatchedDbs] = useState<SectionState>({
    databases: [],
    loading: false,
  });
  const [unmatchedDbs, setUnmatchedDbs] = useState<SectionState>({
    databases: [],
    loading: false,
  });
  const [matchedToken, setMatchedToken] = useState(0);
  const [unmatchedToken, setUnmatchedToken] = useState("");
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set()
  );

  // Refs for stable references across closures
  const matchedNamesRef = useRef<string[]>([]);
  matchedNamesRef.current = matchedNames;

  const pageSize = getDefaultPagination();

  const toggleSection = useCallback((name: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }, []);

  const loadMoreMatched = useCallback(
    async (currentToken: number, currentDbs: Database[]) => {
      const names = matchedNamesRef.current;
      const next = currentToken + pageSize;
      const slice = names.slice(currentToken, next);
      if (slice.length === 0) return;

      setMatchedDbs((prev) => ({ ...prev, error: undefined, loading: true }));
      try {
        await useAppStore.getState().batchGetOrFetchDatabases(slice);
        const newDbs = slice
          .map((n) => useAppStore.getState().getDatabaseByName(n))
          .filter((db) => isValidDatabaseName(db.name));
        setMatchedDbs({
          databases: [...currentDbs, ...newDbs],
          loading: false,
        });
        setMatchedToken(next);
      } catch (error) {
        setMatchedDbs((prev) => ({
          ...prev,
          error: error instanceof Error ? error.message : String(error),
          loading: false,
        }));
      }
    },
    [pageSize]
  );

  const loadMoreUnmatched = useCallback(
    async (token: string, currentDbs: Database[]) => {
      const names = matchedNamesRef.current;
      setUnmatchedDbs((prev) => ({
        ...prev,
        error: undefined,
        loading: true,
      }));
      try {
        let unmatched: Database[] = [];
        let pageToken = token;
        while (true) {
          const { databases, nextPageToken } = await useAppStore
            .getState()
            .fetchDatabases({
              pageToken,
              pageSize,
              parent: project,
              silent: true,
            });
          pageToken = nextPageToken;
          unmatched = databases.filter((db) => !names.includes(db.name));
          if (unmatched.length > 0 || !pageToken) {
            break;
          }
        }
        const validDbs = unmatched.filter((db) => isValidDatabaseName(db.name));
        setUnmatchedDbs({
          databases: [...currentDbs, ...validDbs],
          loading: false,
        });
        setUnmatchedToken(pageToken);
      } catch (error) {
        setUnmatchedDbs((prev) => ({
          ...prev,
          error: error instanceof Error ? error.message : String(error),
          loading: false,
        }));
      }
    },
    [pageSize, project]
  );

  // Debounced expression validation and data loading
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  useEffect(() => {
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(async () => {
      if (!validateSimpleExpr(expr)) {
        setMatchingError(undefined);
        setMatchedNames([]);
        setMatchedDbs({ databases: [], loading: false });
        setUnmatchedDbs({ databases: [], loading: false });
        setMatchedToken(0);
        setUnmatchedToken("");
        setExpandedSections(new Set());
        return;
      }

      setLoading(true);
      try {
        const names = presetNames
          ? presetNames
          : await useAppStore.getState().fetchDatabaseGroupMatchList({
              projectName: project,
              expr,
            });

        setMatchingError(undefined);
        matchedNamesRef.current = names;
        setMatchedNames(names);

        // Reset and load initial batches
        const matchedNext = Math.min(pageSize, names.length);
        const matchedSlice = names.slice(0, matchedNext);

        let newMatchedDbs: Database[] = [];
        if (matchedSlice.length > 0) {
          await useAppStore.getState().batchGetOrFetchDatabases(matchedSlice);
          newMatchedDbs = matchedSlice
            .map((n) => useAppStore.getState().getDatabaseByName(n))
            .filter((db) => isValidDatabaseName(db.name));
        }
        setMatchedDbs({
          databases: newMatchedDbs,
          error: undefined,
          loading: false,
        });
        setMatchedToken(matchedNext);

        // Load unmatched
        let unmatchedList: Database[] = [];
        let pageToken = "";
        while (true) {
          const { databases, nextPageToken } = await useAppStore
            .getState()
            .fetchDatabases({
              pageToken,
              pageSize,
              parent: project,
              silent: true,
            });
          pageToken = nextPageToken;
          unmatchedList = databases.filter((db) => !names.includes(db.name));
          if (unmatchedList.length > 0 || !pageToken) {
            break;
          }
        }
        const validUnmatched = unmatchedList.filter((db) =>
          isValidDatabaseName(db.name)
        );
        setUnmatchedDbs({
          databases: validUnmatched,
          error: undefined,
          loading: false,
        });
        setUnmatchedToken(pageToken);

        // Auto-expand sections that have data
        const expanded = new Set<string>();
        if (newMatchedDbs.length > 0) expanded.add("matched");
        if (validUnmatched.length > 0) expanded.add("unmatched");
        setExpandedSections(expanded);
      } catch (error) {
        setMatchingError(
          error instanceof Error ? error.message : String(error)
        );
      } finally {
        setLoading(false);
      }
    }, DEBOUNCE_SEARCH_DELAY);
    return () => clearTimeout(timerRef.current);
  }, [expr, project, presetNames, pageSize]);

  const hasMoreMatched = matchedToken < matchedNames.length;
  const hasMoreUnmatched = !!unmatchedToken;

  const sections = useMemo(
    () => [
      {
        name: "matched",
        title: t("database-group.matched-databases"),
        databases: matchedDbs.databases,
        error: matchedDbs.error,
        sectionLoading: matchedDbs.loading,
        totalLabel: String(matchedNames.length),
        hasMore: hasMoreMatched,
        onLoadMore: () => loadMoreMatched(matchedToken, matchedDbs.databases),
      },
      {
        name: "unmatched",
        title: t("database-group.unmatched-databases"),
        databases: unmatchedDbs.databases,
        error: unmatchedDbs.error,
        sectionLoading: unmatchedDbs.loading,
        hasMore: hasMoreUnmatched,
        totalLabel: hasMoreUnmatched
          ? t("database-group.unmatched-databases-preview")
          : String(unmatchedDbs.databases.length),
        onLoadMore: () =>
          loadMoreUnmatched(unmatchedToken, unmatchedDbs.databases),
      },
    ],
    [
      t,
      matchedDbs,
      unmatchedDbs,
      matchedNames.length,
      hasMoreMatched,
      hasMoreUnmatched,
      matchedToken,
      unmatchedToken,
      loadMoreMatched,
      loadMoreUnmatched,
    ]
  );

  return (
    <div>
      <div className="mb-2 flex flex-row items-center">
        <span className="font-medium text-main mr-2">
          {t("common.databases")}
        </span>
        {loading && (
          <svg
            className="animate-spin size-5 opacity-60"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
            />
          </svg>
        )}
      </div>

      {matchingError && (
        <p className="my-2 text-sm border border-error px-2 py-1 rounded-sm bg-error/10 text-error">
          {matchingError}
        </p>
      )}

      <div className="border rounded-sm overflow-hidden">
        {sections.map((section) => {
          const isExpanded = expandedSections.has(section.name);
          const isEmpty = section.databases.length === 0;

          return (
            <div key={section.name}>
              <button
                type="button"
                className={`w-full flex items-center justify-between py-2 px-2 text-left text-sm border-b border-control-border/60 last:border-b-0 ${
                  isEmpty
                    ? "cursor-default text-control-placeholder"
                    : "cursor-pointer hover:bg-control-bg"
                }`}
                disabled={isEmpty}
                onClick={() => toggleSection(section.name)}
              >
                <div className="flex items-center gap-x-1">
                  <ChevronRight
                    className={`size-4 transition-transform ${
                      isExpanded ? "rotate-90" : ""
                    } ${isEmpty ? "text-control-border" : ""}`}
                  />
                  <span>{section.title}</span>
                </div>
                {section.totalLabel && (
                  <span className="text-control-light text-xs">
                    {section.totalLabel}
                  </span>
                )}
              </button>

              {isExpanded && !isEmpty && (
                <div className="flex flex-col gap-y-2 w-full max-h-48 overflow-y-auto border-b border-control-border/60 last:border-b-0">
                  <div className="p-1">
                    {section.databases.map((database) => {
                      return (
                        <div
                          key={database.name}
                          className="w-full min-w-0 rounded-xs px-2 py-1.5 hover:bg-control-bg"
                        >
                          <DatabaseTargetDisplay
                            showEnvironment
                            target={database.name}
                          />
                        </div>
                      );
                    })}
                  </div>
                  {section.error && (
                    <div className="mx-2 rounded-sm border border-error bg-error/10 px-2 py-1 text-sm text-error">
                      {t("database-group.load-database-failed")}
                    </div>
                  )}
                  {section.hasMore && (
                    <button
                      type="button"
                      className="self-start px-2 pb-2 text-sm text-accent hover:text-accent/80 disabled:opacity-50"
                      disabled={section.sectionLoading}
                      onClick={section.onLoadMore}
                    >
                      {section.sectionLoading ? "..." : t("common.load-more")}
                    </button>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
