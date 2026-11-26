import type { MouseEventHandler } from "react";

/**
 * A stand-in for `HumanizeTs` that keeps what a test might be about: the
 * instant, and the form the surface asked for. A stub that renders only its
 * children, or nothing, cannot see a surface given the wrong form — which is
 * the decision docs/design/timestamp-display.md exists to make.
 */
export const humanizeTsStub = () => ({
  HumanizeTs: ({
    tsMs,
    mode = "queue",
    precision = "second",
    className,
    truncate = false,
    link,
  }: {
    tsMs: number | undefined;
    mode?: string;
    precision?: string;
    className?: string;
    truncate?: boolean;
    link?: { href: string; onClick?: MouseEventHandler<HTMLAnchorElement> };
  }) => {
    // Nothing at all without an instant, as the component does: a stub that
    // left an empty box behind would hide a row that kept its separator.
    if (tsMs === undefined) return null;
    const props = {
      "data-testid": "humanize-ts",
      "data-mode": mode,
      "data-precision": precision,
      "data-truncate": truncate,
      className,
    };
    // A linked timestamp stays a link, so a test can read where it points.
    return link ? (
      <a {...props} href={link.href} onClick={link.onClick}>
        {tsMs}
      </a>
    ) : (
      <span {...props}>{tsMs}</span>
    );
  },
});

const shown = (root: ParentNode) =>
  Array.from(root.querySelectorAll<HTMLElement>("[data-testid=humanize-ts]"));

/** The form each timestamp was asked for, in document order. */
export const shownTimestampModes = (root: ParentNode) =>
  shown(root).map((node) => node.dataset.mode);

/** The finest unit each timestamp's age was asked to count in, in document order. */
export const shownTimestampPrecisions = (root: ParentNode) =>
  shown(root).map((node) => node.dataset.precision);

/** The instant each timestamp was given, in document order. */
export const shownTimestampInstants = (root: ParentNode) =>
  shown(root).map((node) => node.textContent);

/** Whether each timestamp was asked to fit its box, in document order. */
export const shownTimestampTruncates = (root: ParentNode) =>
  shown(root).map((node) => node.dataset.truncate === "true");
