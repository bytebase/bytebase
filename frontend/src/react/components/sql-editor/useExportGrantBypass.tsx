import { type ReactNode, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { RouterLink } from "@/react/components/RouterLink";
import { useAppStore } from "@/react/stores/app";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";

interface UseExportGrantBypassArgs {
  /**
   * Whether to actively look for a JIT export grant. Pass
   * `policy.disableExport` — when the policy already allows export
   * there's no bypass to surface.
   */
  enabled: boolean;
  /**
   * Project parent for the access-grant search (e.g. `"projects/foo"`).
   * The hook is a no-op when this is empty / undefined.
   */
  project: string | undefined;
  /** Exact statement to match against grant.payload.query. */
  statement: string;
  /**
   * Queried target database resource names. Single-element OK —
   * `ResultView` passes `[database.name]`, `BatchQuerySelect` passes
   * the full queried set. Empty array == no lookup.
   *
   * Internally the hook fires one `searchMyAccessGrants` call per
   * target in parallel (each with `target == "x"` and `pageSize: 1`).
   * The fan-out shape is correct by construction: a single multi-
   * target `target in [...]` query with a row-limit could cluster all
   * results on one target and silently hide coverage for the others.
   */
  targets: readonly string[];
}

interface UseExportGrantBypassResult {
  /**
   * Subset of `targets` for which an active export grant exists.
   * `BatchQuerySelect` uses this to filter the export drawer's
   * `DatabaseTableView` so users can only pick authorized databases.
   * Single-target callers can check `length > 0`.
   */
  matchedDatabases: string[];
  /**
   * Subset of `targets` NOT covered by any grant. Drives the
   * "Request Export" affordance — pre-seed it with these databases.
   */
  unmatchedDatabases: string[];
  /**
   * Fullname of the primary matching grant (first unique grant covering
   * any matched DB), or `""` when none. Callers typically use
   * `!!grantName` to flip `showExport`.
   */
  grantName: string;
  /**
   * Pre-rendered tooltip describing the matched grant(s):
   * - 1 unique grant: title + reason + "View issue" link.
   * - N>1 unique grants: count header + bulleted list of issue titles
   *   (each links to its issue), capped at 5 rows + "… and N more".
   * `undefined` when no grant matched.
   */
  tooltip: ReactNode;
}

// Cap the multi-grant list to keep the tooltip compact. 20 distinct
// grants on a 224px-wide popup would overflow most viewports
// vertically; capping at 5 + "and N more" keeps the worst case
// readable.
const LIST_CAP = 5;

// Cap parallel `SearchMyAccessGrants` lookups so a batch query over
// hundreds of databases (database-group expansion, multi-tenant
// projects) doesn't fire hundreds of concurrent RPCs the moment
// results render — that would put real DB/CEL-parser pressure on the
// API for a UI affordance the user might never interact with. Bot
// review #3357335452.
//
// 10 is a pragmatic ceiling: the common 1–10 DB batch fits in one
// chunk (same single round-trip as before the cap), and a 200-DB
// batch becomes 20 sequential rounds of 10 — bounded server load,
// ≤2s tail latency on typical RTTs.
const LOOKUP_CONCURRENCY = 10;

export function useExportGrantBypass({
  enabled,
  project,
  statement,
  targets,
}: UseExportGrantBypassArgs): UseExportGrantBypassResult {
  const { t } = useTranslation();
  const searchMyAccessGrants = useAppStore((s) => s.searchMyAccessGrants);
  const fetchIssueByName = useAppStore((s) => s.fetchIssueByName);

  // Per-target grant: `grantsByTarget[target]` is the active export
  // grant for that target, or `undefined` when none.
  const [grantsByTarget, setGrantsByTarget] = useState<
    Record<string, AccessGrant | undefined>
  >({});
  // Issue title per unique grant name. Populated by parallel fetches
  // once `uniqueGrants` is known.
  const [issueTitlesByGrantName, setIssueTitlesByGrantName] = useState<
    Record<string, string>
  >({});

  // Stable join key so identity changes on the `targets` array don't
  // re-fire the search effect on every parent render.
  const targetsKey = useMemo(() => targets.join(","), [targets]);

  useEffect(() => {
    if (!enabled || !statement || !project || targetsKey === "") {
      // Idempotent reset — only swap state when it actually needs to
      // clear, so callers passing a fresh `[database.name]` literal
      // each render don't trip a "reset → re-render → reset" loop
      // (`Object.is({}, {})` is false).
      setGrantsByTarget((prev) => (Object.keys(prev).length === 0 ? prev : {}));
      return;
    }
    // Clear stale matches synchronously so the UI doesn't surface a
    // grant matched to the previous (statement, targets) tuple while
    // the new search is in flight. Without this, a user changing the
    // SQL statement would briefly see Export promise authorization
    // the new statement doesn't actually have. Bot review #3357266207.
    // Same idempotent-reset guard so the no-prior-data case (initial
    // mount, subsequent identity-only re-renders) doesn't churn.
    setGrantsByTarget((prev) => (Object.keys(prev).length === 0 ? prev : {}));
    let canceled = false;
    void (async () => {
      // Fan out: one search per target. Each call narrows to the
      // single target server-side so `pageSize: 1` suffices.
      //
      // Each request is independently try/catch'd: a failure on
      // target A (network, auth, rate-limit) MUST NOT discard the
      // successful results from targets B/C/D. `Promise.all` would
      // reject the whole batch on the first failure; we want partial
      // success. The failed target falls back to `undefined`, which
      // ResultView/BatchQuerySelect treat as "no grant" → Request
      // Export surfaces for that DB. Bot review #3357266207.
      //
      // Chunked at `LOOKUP_CONCURRENCY` so we cap the in-flight RPC
      // count even on batches of hundreds of DBs. Bot review
      // #3357335452.
      const byTarget: Record<string, AccessGrant | undefined> = {};
      for (let i = 0; i < targets.length; i += LOOKUP_CONCURRENCY) {
        if (canceled) return;
        const chunk = targets.slice(i, i + LOOKUP_CONCURRENCY);
        const chunkResults = await Promise.all(
          chunk.map(async (target) => {
            try {
              const res = await searchMyAccessGrants({
                parent: project,
                filter: {
                  statementExact: statement,
                  status: ["ACTIVE"],
                  export: true,
                  target,
                },
                pageSize: 1,
              });
              return { target, grant: res.accessGrants[0] };
            } catch {
              return {
                target,
                grant: undefined as AccessGrant | undefined,
              };
            }
          })
        );
        for (const { target, grant } of chunkResults) {
          byTarget[target] = grant;
        }
      }
      if (canceled) return;
      // Single commit at the end (not per-chunk) — the matched /
      // unmatched derivation downstream is whole-set, so partial
      // updates would flicker Export ↔ Request Export as chunks
      // resolve.
      setGrantsByTarget(byTarget);
    })();
    return () => {
      canceled = true;
    };
    // `targets` intentionally NOT in the dep array — the array literal
    // is a fresh identity on every parent render, which would defeat
    // the `targetsKey` stabilization and trigger an infinite "search
    // → setState → re-render → search" loop. The closure captures the
    // `targets` from the render that last bumped `targetsKey`; since
    // `targetsKey` is a value-stable join of the same content, that
    // captured array carries the right values to iterate.
  }, [enabled, project, statement, targetsKey, searchMyAccessGrants]);

  // Partition into matched/unmatched AND build the dedup'd grant list.
  // A single grant can cover multiple targets (its `targets` array
  // overlapping the queried set), so the per-target search returns the
  // same grant for several keys — collapse by `grant.name`.
  //
  // Keyed by `targetsKey` (not `targets`) so per-render fresh array
  // literals don't churn this memo's identity, which would in turn
  // churn every downstream `useMemo` / `useEffect` that depends on
  // `uniqueGrants` (and trip an infinite loop via the issue-fetch
  // effect).
  const { matchedDatabases, unmatchedDatabases, uniqueGrants } = useMemo(() => {
    const matched: string[] = [];
    const unmatched: string[] = [];
    const seen = new Map<string, AccessGrant>();
    for (const target of targets) {
      const grant = grantsByTarget[target];
      if (grant) {
        matched.push(target);
        if (!seen.has(grant.name)) seen.set(grant.name, grant);
      } else {
        unmatched.push(target);
      }
    }
    return {
      matchedDatabases: matched,
      unmatchedDatabases: unmatched,
      uniqueGrants: Array.from(seen.values()),
    };
    // See the search-effect dep-array comment: keyed by `targetsKey`
    // (a value-stable join), not by `targets` (a fresh identity each
    // render).
  }, [targetsKey, grantsByTarget]);

  const primaryGrant = uniqueGrants[0];
  const grantName = primaryGrant?.name ?? "";

  // Stable key over the unique-grant set's `issue` fields so we don't
  // refetch on every identity change to `uniqueGrants`.
  const issueFetchKey = useMemo(
    () => uniqueGrants.map((g) => `${g.name}:${g.issue}`).join("|"),
    [uniqueGrants]
  );

  // Fetch every unique grant's approval-issue title in parallel.
  // `fetchIssueByName` is stateless (no per-issue cache) so the titles
  // live in local state. Keyed by `issueFetchKey` (a value-stable
  // string over the unique-grant set's name+issue pairs); `uniqueGrants`
  // is read from closure to avoid the per-render identity churn that
  // caused the previous infinite loop.
  useEffect(() => {
    if (issueFetchKey === "") {
      // Idempotent reset (`Object.is({}, {})` is false → without this,
      // each render with no unique grants schedules a new render).
      setIssueTitlesByGrantName((prev) =>
        Object.keys(prev).length === 0 ? prev : {}
      );
      return;
    }
    let canceled = false;
    void (async () => {
      const withIssue = uniqueGrants.filter((g) => g.issue);
      if (withIssue.length === 0) {
        setIssueTitlesByGrantName((prev) =>
          Object.keys(prev).length === 0 ? prev : {}
        );
        return;
      }
      const fetched = await Promise.all(
        withIssue.map(async (g) => {
          try {
            const issue = await fetchIssueByName(g.issue, true /* silent */);
            return { name: g.name, title: issue?.title ?? "" };
          } catch {
            return { name: g.name, title: "" };
          }
        })
      );
      if (canceled) return;
      const map: Record<string, string> = {};
      for (const { name, title } of fetched) {
        if (title) map[name] = title;
      }
      setIssueTitlesByGrantName(map);
    })();
    return () => {
      canceled = true;
    };
  }, [issueFetchKey, fetchIssueByName]);

  const tooltip = useMemo<ReactNode>(() => {
    if (!enabled || uniqueGrants.length === 0) return undefined;

    // ============ Single-grant detail view ============
    if (uniqueGrants.length === 1) {
      const grant = uniqueGrants[0];
      const fetchedTitle = issueTitlesByGrantName[grant.name];
      // Title format: `Export available via issue "X"` when the grant
      // has an approval issue whose title we could fetch; otherwise
      // the generic phrase. The generic phrase also covers the brief
      // window before the issue fetch resolves.
      const headline = fetchedTitle
        ? t("sql-editor.export-available-via-issue", { title: fetchedTitle })
        : t("sql-editor.export-enabled-by-grant");
      const issueName = grant.issue ?? "";
      const issueHref = issueName
        ? issueName.startsWith("/")
          ? issueName
          : `/${issueName}`
        : undefined;
      // No min-width here — that would override the primitive
      // Tooltip's `max-w-56` cap (min-width wins over max-width),
      // pushing the popup past the viewport when the trigger is near
      // the right edge and Floating UI can't shift it back.
      return (
        <div className="flex flex-col gap-y-1">
          {/* `break-words` wraps long emails/IDs in the title onto a
              new line instead of clipping. */}
          <span className="font-medium break-words">{headline}</span>
          {grant.reason && (
            // Cap at 3 lines with auto-ellipsis. Full reason is one
            // click away via the View Issue link.
            <span className="text-xs opacity-80 break-words line-clamp-3">
              {grant.reason}
            </span>
          )}
          {issueHref && (
            <div className="flex justify-end pt-1">
              <RouterLink
                to={issueHref}
                target="_blank"
                rel="noreferrer"
                className="text-xs underline whitespace-nowrap"
                onClick={(e) => e.stopPropagation()}
              >
                {t("sql-editor.view-issue")}
              </RouterLink>
            </div>
          )}
        </div>
      );
    }

    // ============ Multi-grant list view (N>1) ============
    const visible = uniqueGrants.slice(0, LIST_CAP);
    const overflow = uniqueGrants.length - visible.length;
    return (
      <div className="flex flex-col gap-y-1">
        <span className="font-medium break-words">
          {t("sql-editor.export-available-via-n-grants", {
            count: uniqueGrants.length,
          })}
        </span>
        <div className="flex flex-col gap-y-0.5 text-xs">
          {visible.map((grant) => {
            const issueName = grant.issue ?? "";
            const href = issueName
              ? issueName.startsWith("/")
                ? issueName
                : `/${issueName}`
              : undefined;
            // Display priority: fetched issue title → grant.reason →
            // generic phrase. `reason` exists on every grant the user
            // submitted; the generic phrase only kicks in for auto-
            // active grants without an issue or reason, which are
            // rare. `line-clamp-1` truncates long titles/reasons so
            // each row is exactly one line.
            const display =
              issueTitlesByGrantName[grant.name] ||
              grant.reason ||
              t("sql-editor.export-enabled-by-grant");
            return (
              <div key={grant.name} className="flex items-start gap-x-1">
                <span className="select-none opacity-60">•</span>
                {href ? (
                  <RouterLink
                    to={href}
                    target="_blank"
                    rel="noreferrer"
                    className="flex-1 min-w-0 underline break-words line-clamp-1"
                    onClick={(e) => e.stopPropagation()}
                  >
                    {display}
                  </RouterLink>
                ) : (
                  <span className="flex-1 min-w-0 break-words line-clamp-1">
                    {display}
                  </span>
                )}
              </div>
            );
          })}
          {overflow > 0 && (
            <div className="opacity-70 italic">
              {overflow === 1
                ? t("sql-editor.and-more-grant", { count: overflow })
                : t("sql-editor.and-more-grants", { count: overflow })}
            </div>
          )}
        </div>
      </div>
    );
  }, [t, enabled, uniqueGrants, issueTitlesByGrantName]);

  return {
    matchedDatabases,
    unmatchedDatabases,
    grantName,
    tooltip,
  };
}
