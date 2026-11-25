import { useLayoutEffect, useRef } from "react";
import { useActivityScope, useCommentLinks } from "../shell/CommentLinkContext";

// The page's sticky title row covers this much of the top of the scroll pane.
const STICKY_HEADER_HEIGHT = 64;

// Brings an Activity entry to the reader: into view when it is not already
// there, without animation, and under keyboard focus so the next Tab starts
// from it. An entry already in view stays where it is.
function landOn(element: HTMLElement) {
  const pane = element.closest<HTMLElement>("[data-scroll-restoration-id]");
  const view = pane?.getBoundingClientRect() ?? {
    top: 0,
    bottom: window.innerHeight,
  };
  const rect = element.getBoundingClientRect();
  const inView =
    rect.top >= view.top + STICKY_HEADER_HEIGHT && rect.bottom <= view.bottom;
  if (!inView) element.scrollIntoView({ block: "center" });
  if (!element.hasAttribute("tabindex")) element.tabIndex = -1;
  element.focus({ preventScroll: true });
}

// For an element in the Activity timeline that stands for the given comments:
// whether the page's address names one of them, and a ref that takes the
// landing when the timeline is asked to show one. Outside the timeline both
// are inert.
export function useActivityLanding<T extends HTMLElement>(
  commentNames: readonly string[]
) {
  const scope = useActivityScope();
  const linkedName = useCommentLinks()?.linkedCommentName;
  const ref = useRef<T>(null);
  const focus =
    scope?.focus && commentNames.includes(scope.focus.commentName)
      ? scope.focus
      : undefined;
  const onLanded = scope?.onLanded;
  useLayoutEffect(() => {
    if (!focus || !ref.current) return;
    landOn(ref.current);
    onLanded?.(focus.nonce);
  }, [focus, onLanded]);
  const linked =
    scope !== undefined &&
    linkedName !== undefined &&
    commentNames.includes(linkedName);
  return { linked, ref };
}

// The comment the timeline is asked to show, when it is one of the given
// ones: a card has to unfold before that comment can take the landing.
export function useActivityReveal(
  commentNames: readonly string[]
): string | undefined {
  const focus = useActivityScope()?.focus;
  return focus && commentNames.includes(focus.commentName)
    ? focus.commentName
    : undefined;
}
