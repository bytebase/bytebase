// Single source of truth for review/activity icons (BYT-9713).
//
// The same action must look the same everywhere it appears — the button you
// click to approve/reject and the badge that act produces in the activity
// timeline. This module declares the canonical icon + color for each action and
// activity event; every surface consumes these specs instead of hard-coding its
// own glyph, so a change here updates all surfaces at once.
//
// Tone rule (applied consistently across every family):
//   success → positive / added / done   (green)
//   error   → negative / deleted / rejected (red)
//   neutral → edits & process            (gray)
//
// Icon families:
//   Review decision (action + activity): approve 👍 / reject 👎 / re-request ↺
//   Plan change (activity):  added / deleted / edited — one symmetric "file"
//     trio, so "edit a change" (FilePen) never collides with an issue-field
//     edit (Pencil) or a reject.
//   Issue event (activity):  field edit (Pencil) / resolved (CheckCircle2)
//
// `Pencil` is reserved for exactly one meaning — an issue *field* edit.
import {
  CheckCircle2,
  FileMinus2,
  FilePen,
  FilePlus2,
  type LucideIcon,
  Pencil,
  RotateCcw,
  ThumbsDown,
  ThumbsUp,
} from "lucide-react";

export type IconTone = "success" | "error" | "neutral";

export interface ActivityIconSpec {
  Icon: LucideIcon;
  tone: IconTone;
}

// Review decisions — the only family rendered on two surfaces (the action
// buttons and the activity badges), so they share one descriptor.
export const REVIEW_DECISION_ICON = {
  approved: { Icon: ThumbsUp, tone: "success" },
  rejected: { Icon: ThumbsDown, tone: "error" },
  reRequested: { Icon: RotateCcw, tone: "neutral" },
} satisfies Record<string, ActivityIconSpec>;

// Plan change events — activity only.
export const PLAN_CHANGE_ICON = {
  added: { Icon: FilePlus2, tone: "success" },
  deleted: { Icon: FileMinus2, tone: "error" },
  edited: { Icon: FilePen, tone: "neutral" },
} satisfies Record<string, ActivityIconSpec>;

// Issue events — activity only.
export const ISSUE_EVENT_ICON = {
  fieldEdit: { Icon: Pencil, tone: "neutral" },
  resolved: { Icon: CheckCircle2, tone: "success" },
} satisfies Record<string, ActivityIconSpec>;

// Tone → classes. Activity badges fill the circle (white glyph on bg-tone);
// action buttons tint just the glyph (text-tone).
export const ICON_BADGE_TONE: Record<IconTone, string> = {
  success: "bg-success text-white",
  error: "bg-error text-white",
  neutral: "bg-control-bg text-control",
};

export const ICON_TEXT_TONE: Record<IconTone, string> = {
  success: "text-success",
  error: "text-error",
  neutral: "text-control",
};
