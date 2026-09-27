// Pure layout math for the horizontal approval flow (spec: "Approval flow
// renderer"). The current (or rejected) step is the anchor and never folds;
// approved steps fold first into one leading chip; trailing pending steps
// fold into one dashed chip, farthest-from-current first.
export type ApprovalStepStatus =
  | "approved"
  | "rejected"
  | "current"
  | "pending";

export type ApprovalFlowLayout =
  | { kind: "vertical" }
  | {
      kind: "horizontal";
      // Leading approved steps folded into the "N approved" chip (0 = all named).
      foldedApproved: number;
      // Pending steps after the anchor rendered as full named nodes,
      // counted from the one nearest the anchor.
      namedPending: number;
    };

export const VERTICAL_BREAKPOINT_PX = 560;

// Width budget per element, connector included. Estimates — the row also has
// min-w-0 truncation, so being a little conservative is fine.
const NAMED_NODE_PX = 190;
const ANCHOR_NODE_PX = 240;
const CHIP_PX = 150;

export function computeApprovalFlowLayout(
  statuses: ApprovalStepStatus[],
  containerWidth: number
): ApprovalFlowLayout {
  if (containerWidth < VERTICAL_BREAKPOINT_PX) {
    return { kind: "vertical" };
  }

  const anchorIndex = statuses.findIndex(
    (s) => s === "current" || s === "rejected"
  );
  // Fully-approved (or skipped) flows have no anchor; render all named if they
  // fit, otherwise fold approved into the chip.
  const approvedCount = anchorIndex === -1 ? statuses.length : anchorIndex;
  const pendingCount =
    anchorIndex === -1 ? 0 : statuses.length - anchorIndex - 1;
  const anchorCost = anchorIndex === -1 ? 0 : ANCHOR_NODE_PX;

  const fits = (foldedApproved: number, namedPending: number): boolean => {
    const namedApproved = approvedCount - foldedApproved;
    const foldedPending = pendingCount - namedPending;
    const width =
      (foldedApproved > 0 ? CHIP_PX : 0) +
      namedApproved * NAMED_NODE_PX +
      anchorCost +
      namedPending * NAMED_NODE_PX +
      (foldedPending > 0 ? CHIP_PX : 0);
    return width <= containerWidth;
  };

  // Name as many nodes as the width allows. Prefer naming upcoming (pending)
  // steps, then the most recent approved ones; both counted nearest the anchor.
  // For each pending count (most first), fold the fewest approved that still
  // fits — so a roomy row shows partial "N approved" + several named nodes
  // instead of collapsing everything into one chip.
  for (let namedPending = pendingCount; namedPending >= 0; namedPending--) {
    for (
      let foldedApproved = 0;
      foldedApproved <= approvedCount;
      foldedApproved++
    ) {
      if (fits(foldedApproved, namedPending)) {
        return { kind: "horizontal", foldedApproved, namedPending };
      }
    }
  }
  // Minimum form regardless: chip + anchor + chip.
  return { kind: "horizontal", foldedApproved: approvedCount, namedPending: 0 };
}
