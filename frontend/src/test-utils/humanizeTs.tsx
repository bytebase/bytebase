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
  }: {
    tsMs: number | undefined;
    mode?: string;
  }) =>
    // Nothing at all without an instant, as the component does: a stub that
    // left an empty box behind would hide a row that kept its separator.
    tsMs === undefined ? null : (
      <span data-testid="humanize-ts" data-mode={mode}>
        {tsMs}
      </span>
    ),
});
