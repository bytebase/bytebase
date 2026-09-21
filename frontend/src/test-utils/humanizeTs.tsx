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
    className,
  }: {
    tsMs: number | undefined;
    mode?: string;
    className?: string;
  }) =>
    // Nothing at all without an instant, as the component does: a stub that
    // left an empty box behind would hide a row that kept its separator. The
    // class lands on the same span it does in the component, since that is
    // where a caller asks a narrowed date to ellipsize rather than wrap.
    tsMs === undefined ? null : (
      <span data-testid="humanize-ts" data-mode={mode} className={className}>
        {tsMs}
      </span>
    ),
});

const shown = (root: ParentNode) =>
  Array.from(root.querySelectorAll<HTMLElement>("[data-testid=humanize-ts]"));

/** The form each timestamp was asked for, in document order. */
export const shownTimestampModes = (root: ParentNode) =>
  shown(root).map((node) => node.dataset.mode);

/** The instant each timestamp was given, in document order. */
export const shownTimestampInstants = (root: ParentNode) =>
  shown(root).map((node) => node.textContent);
