import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { StageSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanLifecycleHeaderState } from "./planLifecycleHeaderState";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("../../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({ issue: { name: "projects/p/issues/1" } }),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
}));

vi.mock("./LifecycleStamp", () => ({
  LifecycleStamp: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
}));

vi.mock("./ReviewYourTurnAction", () => ({
  ReviewYourTurnAction: () => <div>marker-review-your-turn</div>,
}));

vi.mock("./PlanStatusAction", () => ({
  PlanStatusAction: () => <div>marker-plan-status</div>,
}));

vi.mock("./DeployStageActions", () => ({
  RunStageAction: () => <div>marker-run-stage</div>,
  FrontierStatusStamp: () => <div>marker-running-stage</div>,
}));

import React from "react";
import { PlanLifecycleSlot } from "./PlanLifecycleSlot";
import { PlanLifecycleStamp } from "./PlanLifecycleStamp";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  document.body.removeChild(container);
});

const renderEl = (element: React.ReactElement) =>
  act(() => {
    root.render(element);
  });

const slot = (state: PlanLifecycleHeaderState) =>
  renderEl(<PlanLifecycleSlot state={state} />);

const stage = create(StageSchema, { name: "stages/1" });

describe("PlanLifecycleSlot (right action area)", () => {
  test("renders the matching control for each active state", () => {
    slot({ kind: "review-generating" });
    expect(container.textContent).toContain("plan.review.action");

    slot({ kind: "review-your-turn" });
    expect(container.textContent).toContain("marker-review-your-turn");

    slot({ kind: "plan-status", reason: "rejected" });
    expect(container.textContent).toContain("marker-plan-status");

    slot({ kind: "preparing-rollout" });
    expect(container.textContent).toContain("plan.lifecycle.preparing-rollout");

    slot({ kind: "run-stage", stage });
    expect(container.textContent).toContain("marker-run-stage");

    slot({ kind: "running-stage", stage });
    expect(container.textContent).toContain("marker-running-stage");
  });

  test("renders nothing for header-owned, terminal, and none states", () => {
    // create / ready-for-review render in the header; closed / deployed render
    // as the left stamp; none renders nothing.
    for (const kind of [
      "create",
      "ready-for-review",
      "closed",
      "deployed",
      "none",
    ] as const) {
      slot({ kind });
      expect(container.textContent).toBe("");
    }
  });
});

describe("PlanLifecycleStamp (left of title)", () => {
  test("renders the terminal stamp for closed / deployed", () => {
    renderEl(<PlanLifecycleStamp state={{ kind: "closed" }} />);
    expect(container.textContent).toContain("common.closed");

    renderEl(<PlanLifecycleStamp state={{ kind: "deployed" }} />);
    expect(container.textContent).toContain("plan.lifecycle.deployed");
  });

  test("renders nothing for non-terminal states", () => {
    for (const kind of [
      "create",
      "review-your-turn",
      "running-stage",
    ] as const) {
      renderEl(
        <PlanLifecycleStamp
          state={kind === "running-stage" ? { kind, stage } : { kind }}
        />
      );
      expect(container.textContent).toBe("");
    }
  });
});
