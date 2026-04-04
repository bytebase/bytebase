import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { validateSimpleExpr } from "@/plugins/cel";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import { DEBOUNCE_SEARCH_DELAY, isValidDatabaseName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  getDatabaseEnvironment,
  getDefaultPagination,
  getInstanceResource,
} from "@/utils";

interface MatchedDatabaseViewProps {
  project: string;
  expr: ConditionGroupExpr;
  matchedDatabaseNames?: string[];
}

interface SectionState {
  databases: Database[];
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
  const dbGroupStore = useDBGroupStore();
  const databaseStore = useDatabaseV1Store();

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

      setMatchedDbs((prev) => ({ ...prev, loading: true }));
      try {
        await databaseStore.batchGetOrFetchDatabases(slice);
        const newDbs = slice
          .map((n) => databaseStore.getDatabaseByName(n))
          .filter((db) => isValidDatabaseName(db.name));
        setMatchedDbs({
          databases: [...currentDbs, ...newDbs],
          loading: false,
        });
        setMatchedToken(next);
      } catch {
        setMatchedDbs((prev) => ({ ...prev, loading: false }));
      }
    },
    [databaseStore, pageSize]
  );

  const loadMoreUnmatched = useCallback(
    async (token: string, currentDbs: Database[]) => {
      const names = matchedNamesRef.current;
      setUnmatchedDbs((prev) => ({ ...prev, loading: true }));
      try {
        let unmatched: Database[] = [];
        let pageToken = token;
        while (true) {
          const { databases, nextPageToken } =
            await databaseStore.fetchDatabases({
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
      } catch {
        setUnmatchedDbs((prev) => ({ ...prev, loading: false }));
      }
    },
    [databaseStore, pageSize, project]
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
          : await dbGroupStore.fetchDatabaseGroupMatchList({
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
          await databaseStore.batchGetOrFetchDatabases(matchedSlice);
          newMatchedDbs = matchedSlice
            .map((n) => databaseStore.getDatabaseByName(n))
            .filter((db) => isValidDatabaseName(db.name));
        }
        setMatchedDbs({ databases: newMatchedDbs, loading: false });
        setMatchedToken(matchedNext);

        // Load unmatched
        let unmatchedList: Database[] = [];
        let pageToken = "";
        while (true) {
          const { databases, nextPageToken } =
            await databaseStore.fetchDatabases({
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
        setUnmatchedDbs({ databases: validUnmatched, loading: false });
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
  }, [expr, project, presetNames, dbGroupStore, databaseStore, pageSize]);

  const hasMoreMatched = matchedToken < matchedNames.length;
  const hasMoreUnmatched = !!unmatchedToken;

  const sections = useMemo(
    () => [
      {
        name: "matched",
        title: t("database-group.matched-database"),
        databases: matchedDbs.databases,
        sectionLoading: matchedDbs.loading,
        total: matchedNames.length,
        showTotal: true,
        hasMore: hasMoreMatched,
        onLoadMore: () => loadMoreMatched(matchedToken, matchedDbs.databases),
      },
      {
        name: "unmatched",
        title: t("database-group.unmatched-database"),
        databases: unmatchedDbs.databases,
        sectionLoading: unmatchedDbs.loading,
        hasMore: hasMoreUnmatched,
        showTotal: false,
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
            className="animate-spin h-5 w-5 opacity-60"
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
        <p className="my-2 text-sm border border-red-600 px-2 py-1 rounded-lg bg-red-50 text-red-600">
          {matchingError}
        </p>
      )}

      <div className="border p-2 rounded-lg">
        {sections.map((section) => {
          const isExpanded = expandedSections.has(section.name);
          const isEmpty = section.databases.length === 0;

          return (
            <div key={section.name}>
              <button
                type="button"
                className={`w-full flex items-center justify-between py-2 px-1 text-left text-sm ${
                  isEmpty
                    ? "cursor-default text-gray-400"
                    : "cursor-pointer hover:bg-gray-50"
                }`}
                disabled={isEmpty}
                onClick={() => toggleSection(section.name)}
              >
                <div className="flex items-center gap-x-1">
                  <svg
                    className={`h-4 w-4 transition-transform ${
                      isExpanded ? "rotate-90" : ""
                    } ${isEmpty ? "text-gray-300" : ""}`}
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2}
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M9 5l7 7-7 7"
                    />
                  </svg>
                  <span>{section.title}</span>
                </div>
                {section.showTotal && (
                  <span className="text-gray-500 text-xs">{section.total}</span>
                )}
              </button>

              {isExpanded && !isEmpty && (
                <div className="flex flex-col gap-y-2 w-full max-h-48 overflow-y-auto">
                  <div>
                    {section.databases.map((database) => {
                      const instance = getInstanceResource(database);
                      const env = getDatabaseEnvironment(database);
                      const dbName = database.name.split("/").pop();

                      return (
                        <div
                          key={database.name}
                          className="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
                        >
                          <span className="truncate">{dbName}</span>
                          <div className="flex-1 flex flex-row justify-end items-center shrink-0">
                            <span className="text-sm">{instance.title}</span>
                            <span className="ml-1 text-sm text-gray-400 max-w-31 truncate">
                              {env.title}
                            </span>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                  {section.hasMore && (
                    <button
                      type="button"
                      className="self-start px-2 py-1 text-sm text-blue-600 hover:text-blue-800 disabled:opacity-50"
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
