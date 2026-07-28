// The header's primary lifecycle advance, shared by the two draft states
// (create, ready-for-review).
//
// One rule, three tiers, derived from data rather than hard-coded per state:
//
//   0  nothing unresolved      -> the press advances, and opens nothing
//   1  unmet conditions        -> the press states all of them in a list
//   2  a deliberate override   -> the press opens that one decision
//
// The button is always rendered and always enabled (except in flight), so an
// unmet condition is never a dead control with the explanation locked behind a
// hover. Both surfaces are popovers anchored to the button, so the explanation
// stays attached to the control it belongs to instead of spanning the header.
// The blocker list is information: no fields, no confirm, no links away — and it
// closes itself as the last condition clears.
//
// Callers reset it by navigation: key the element on the plan so a surface
// opened for one plan cannot greet the reader on the next.
import { CircleX, Clock3, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent } from "@/components/ui/popover";
import type { AdvanceBlocker, AdvanceDecision } from "./advanceState";

export interface LifecycleAdvanceProps {
  blockers: AdvanceBlocker[];
  busy?: boolean;
  decision?: AdvanceDecision;
  /** Names the blocked state, e.g. "Cannot create plan". */
  heading: string;
  onAdvance: () => void;
  /** Optional nudge toward an unmet condition, e.g. focus the empty title. */
  onBlocked?: (blockers: AdvanceBlocker[]) => void;
  verb: string;
}

export function LifecycleAdvanceButton({
  blockers,
  busy = false,
  decision,
  heading,
  onAdvance,
  onBlocked,
  verb,
}: LifecycleAdvanceProps) {
  const { t } = useTranslation();
  // The two surfaces are mutually exclusive by construction, so one value
  // rather than two booleans that must never both be true.
  const [surface, setSurface] = useState<"none" | "blockers" | "decision">(
    "none"
  );
  const buttonRef = useRef<HTMLButtonElement>(null);

  // Derived visibility prevents prop changes from briefly rendering a stale
  // tier before the stored surface is reconciled below.
  const showBlockers = surface !== "none" && blockers.length > 0;
  const showDecision =
    surface === "decision" && blockers.length === 0 && decision !== undefined;
  const dismiss = () => setSurface("none");

  // Clear or switch an invalid tier so it cannot reopen without another press.
  useEffect(() => {
    setSurface((current) => {
      if (current === "blockers" && blockers.length === 0) {
        return "none";
      }
      if (current !== "decision") {
        return current;
      }
      if (blockers.length > 0) {
        return "blockers";
      }
      if (!decision) {
        return "none";
      }
      return current;
    });
  }, [blockers.length, decision]);

  const press = () => {
    if (blockers.length > 0) {
      setSurface("blockers");
      onBlocked?.(blockers);
      return;
    }
    if (decision) {
      setSurface("decision");
      return;
    }
    onAdvance();
  };

  return (
    <>
      <Button disabled={busy} onClick={press} ref={buttonRef} type="button">
        {busy && <Loader2 className="mr-2 size-4 animate-spin" />}
        {verb}
      </Button>
      {/* Controlled and anchored rather than trigger-driven: a press opens this
          only at tier 1 or 2, so it must not toggle on every press. */}
      <Popover
        onOpenChange={(open) => {
          if (!open) dismiss();
        }}
        open={showBlockers || showDecision}
      >
        <PopoverContent
          align="end"
          anchor={buttonRef}
          // Sized by its content, not fixed: a one-line blocker on a phone
          // otherwise renders a near-full-bleed panel that reads as detached
          // from the button. The floor keeps it from resizing visibly as
          // blockers resolve; the cap keeps it inside the viewport.
          className="min-w-64 max-w-[min(22rem,calc(100vw-2rem))] p-0"
          initialFocus={false}
        >
          {showBlockers ? (
            <div className="divide-y" role="alert">
              <p className="px-4 py-2.5 text-sm font-medium text-error">
                {heading}
              </p>
              {blockers.map((blocker) => (
                <div
                  className="flex items-start gap-x-2 px-4 py-2.5 text-sm text-control"
                  key={blocker.id}
                >
                  {blocker.kind === "wait" ? (
                    <Clock3 className="mt-0.5 size-4 shrink-0 text-warning" />
                  ) : (
                    <CircleX className="mt-0.5 size-4 shrink-0 text-error" />
                  )}
                  <span className="min-w-0 flex-1">{blocker.message}</span>
                  {blocker.kind === "wait" && (
                    <span className="shrink-0 text-xs text-control-light">
                      {t("plan.blocker.clears-on-its-own")}
                    </span>
                  )}
                </div>
              ))}
            </div>
          ) : (
            decision && (
              <div className="divide-y">
                {/* Warning-toned: the action can still proceed, so this must not
                    read as the error state the blocker list uses. */}
                <p className="px-4 py-2.5 text-sm font-medium text-warning">
                  {decision.headline}
                </p>
                <p className="px-4 py-2.5 text-sm text-control">
                  {decision.body}
                </p>
                <div className="flex gap-x-2 rounded-b-sm bg-control-bg/50 px-4 py-2.5">
                  <Button
                    disabled={busy}
                    onClick={() => {
                      dismiss();
                      onAdvance();
                    }}
                    size="sm"
                    type="button"
                  >
                    {busy && <Loader2 className="size-4 animate-spin" />}
                    {decision.verb}
                  </Button>
                  <Button
                    appearance="outline"
                    onClick={dismiss}
                    size="sm"
                    type="button"
                  >
                    {t("common.cancel")}
                  </Button>
                </div>
              </div>
            )
          )}
        </PopoverContent>
      </Popover>
    </>
  );
}
