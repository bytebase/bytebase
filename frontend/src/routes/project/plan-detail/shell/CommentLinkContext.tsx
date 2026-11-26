import {
  createContext,
  type MouseEvent,
  type ReactNode,
  useContext,
  useMemo,
} from "react";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/app/router/handles";
import type { TimestampLink } from "@/components/HumanizeTs";
import { commentFragmentOf } from "../utils/commentLink";
import { usePlanDetailContext } from "./PlanDetailContext";

export type CommentLinks = {
  // The comment the page's address names, when the plan's issue has it.
  linkedCommentName: string | undefined;
  // The fragment alone, so following it keeps the path the reader is on.
  hrefOf: (commentName: string) => string | undefined;
  // The absolute address on the plan's root path, for copying.
  urlOf: (commentName: string) => string | undefined;
  // Follows a plain click on a comment's link without loading the page.
  follow: (event: MouseEvent<HTMLAnchorElement>, commentName: string) => void;
};

const CommentLinkContext = createContext<CommentLinks | undefined>(undefined);

// Undefined outside a plan page, where a comment has no address to offer.
export const useCommentLinks = () => useContext(CommentLinkContext);

// The link a comment's timestamp becomes, where the comment has an address.
export function useCommentTimestampLink(
  commentName: string | undefined
): TimestampLink | undefined {
  const links = useCommentLinks();
  return useMemo(() => {
    const href = commentName ? links?.hrefOf(commentName) : undefined;
    if (!href || !commentName) return undefined;
    return {
      href,
      onClick: (event) => links?.follow(event, commentName),
    };
  }, [commentName, links]);
}

export type ActivityFocus = { commentName: string; nonce: number };

type ActivityScopeValue = {
  // The comment the timeline is asked to show, until its entry takes it.
  focus: ActivityFocus | undefined;
  onLanded: (nonce: number) => void;
};

const ActivityScopeContext = createContext<ActivityScopeValue | undefined>(
  undefined
);

// Wraps the Activity timeline, the one place a comment's address lands. The
// statement editor draws the same comments outside it, where they neither
// take a landing nor show the outline.
export function ActivityScope({
  children,
  focus,
  onLanded,
}: {
  children: ReactNode;
  focus: ActivityFocus | undefined;
  onLanded: (nonce: number) => void;
}) {
  const value = useMemo(() => ({ focus, onLanded }), [focus, onLanded]);
  return (
    <ActivityScopeContext.Provider value={value}>
      {children}
    </ActivityScopeContext.Provider>
  );
}

export const useActivityScope = () => useContext(ActivityScopeContext);

export function CommentLinkProvider({
  children,
  linkedCommentName,
}: {
  children: ReactNode;
  linkedCommentName: string | undefined;
}) {
  const { bypassLeaveGuardOnce, isEditing, planId, projectId } =
    usePlanDetailContext();
  const value = useMemo<CommentLinks>(() => {
    const hrefOf = (commentName: string) => {
      const fragment = commentFragmentOf(commentName);
      return fragment ? `#${fragment}` : undefined;
    };
    return {
      linkedCommentName,
      hrefOf,
      urlOf: (commentName) => {
        const href = hrefOf(commentName);
        if (!href) return undefined;
        const plan = router.resolve({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: { planId, projectId },
        });
        return `${window.location.origin}${plan.href}${href}`;
      },
      follow: (event, commentName) => {
        if (
          event.defaultPrevented ||
          event.button !== 0 ||
          event.metaKey ||
          event.ctrlKey ||
          event.shiftKey ||
          event.altKey
        ) {
          return;
        }
        const href = hrefOf(commentName);
        if (!href) return;
        event.preventDefault();
        if (window.location.hash === href) return;
        // Only the fragment changes, so nothing being edited is left behind.
        if (isEditing) bypassLeaveGuardOnce();
        void router.push(
          {
            fullPath: `${window.location.pathname}${window.location.search}${href}`,
          },
          { preventScrollReset: true }
        );
      },
    };
  }, [bypassLeaveGuardOnce, isEditing, linkedCommentName, planId, projectId]);
  return (
    <CommentLinkContext.Provider value={value}>
      {children}
    </CommentLinkContext.Provider>
  );
}
