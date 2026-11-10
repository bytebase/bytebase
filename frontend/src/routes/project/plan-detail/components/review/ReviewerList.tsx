import { useVirtualizer } from "@tanstack/react-virtual";
import { type RefObject, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AccountTypeBadge } from "@/components/AccountTypeBadge";
import { getAvatarColor, getInitials } from "@/components/UserAvatar";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { SearchInput } from "@/components/ui/search-input";
import { principalLabel } from "@/hooks/usePrincipal";
import type { ApprovalCandidate } from "../../../useApprovalCandidates";

// A step with more candidates than this gets a search box.
export const REVIEWER_SEARCH_THRESHOLD = 10;

// Rows are measured once mounted; this only sizes rows not yet seen.
const ESTIMATED_ROW_HEIGHT = 44;

const matchesQuery = (candidate: ApprovalCandidate, query: string) =>
  principalLabel(candidate.user).toLowerCase().includes(query) ||
  candidate.user.email.toLowerCase().includes(query);

// The popover body for a step's reviewers: every candidate, searchable by name
// or email, each with its email one click from the clipboard.
export function ReviewerList({
  candidates,
  searchRef,
}: {
  candidates: ApprovalCandidate[];
  searchRef?: RefObject<HTMLInputElement | null>;
}) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);
  const showSearch = candidates.length > REVIEWER_SEARCH_THRESHOLD;
  const matches = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return normalized
      ? candidates.filter((candidate) => matchesQuery(candidate, normalized))
      : candidates;
  }, [candidates, query]);

  const virtualizer = useVirtualizer({
    count: matches.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ESTIMATED_ROW_HEIGHT,
    getItemKey: (index) => matches[index].user.name,
    measureElement: (element) => element.getBoundingClientRect().height,
    overscan: 8,
  });

  return (
    <div className="flex flex-col gap-y-2">
      {showSearch && (
        <SearchInput
          ref={searchRef}
          size="sm"
          value={query}
          placeholder={t("plan.review.approval-flow.search-placeholder")}
          aria-label={t("plan.review.approval-flow.search-placeholder")}
          onChange={(event) => setQuery(event.target.value)}
        />
      )}
      {matches.length === 0 ? (
        <div className="flex flex-col items-center gap-y-2 py-4 text-center text-sm text-control-light">
          <span>
            {t("plan.review.approval-flow.no-matching-reviewers", {
              query: query.trim(),
            })}
          </span>
          <Button
            size="sm"
            appearance="outline"
            onClick={() => {
              setQuery("");
              searchRef?.current?.focus();
            }}
          >
            {t("plan.review.approval-flow.clear-search")}
          </Button>
        </div>
      ) : (
        <div ref={scrollRef} className="max-h-64 overflow-y-auto">
          <div
            role="list"
            className="relative w-full"
            style={{ height: `${virtualizer.getTotalSize()}px` }}
          >
            {virtualizer.getVirtualItems().map((item) => (
              <div
                key={item.key}
                ref={virtualizer.measureElement}
                data-index={item.index}
                role="listitem"
                className="absolute top-0 left-0 w-full pb-2"
                style={{ transform: `translateY(${item.start}px)` }}
              >
                <CandidateRow candidate={matches[item.index]} />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function CandidateRow({ candidate }: { candidate: ApprovalCandidate }) {
  const { t } = useTranslation();
  const { user } = candidate;
  const label = principalLabel(user);
  return (
    <div className="flex items-center gap-x-2">
      <span
        className="flex size-6 shrink-0 items-center justify-center rounded-full font-medium text-accent-text text-xs"
        style={{ backgroundColor: getAvatarColor(label) }}
      >
        {getInitials(label)}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-x-1.5">
          <span data-reviewer-name className="truncate text-sm text-main">
            {label}
          </span>
          <AccountTypeBadge compact email={user.email} />
        </div>
        <div className="truncate text-xs text-control-light">{user.email}</div>
        {candidate.ineligibilities.includes("last-plan-editor") && (
          <div className="text-xs text-warning">
            {t("plan.review.candidate-last-changed-plan")}
          </div>
        )}
      </div>
      <CopyButton
        content={user.email}
        label={t("plan.review.approval-flow.copy-email", { user: label })}
        size="sm"
        className="shrink-0"
      />
    </div>
  );
}
