import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, test } from "vitest";
import { PlanDetailStoreProvider } from "../PlanDetailStoreProvider";
import { usePlanDetailStore } from "../usePlanDetailStore";

const wrapper = ({ children }: { children: ReactNode }) => (
  <PlanDetailStoreProvider>{children}</PlanDetailStoreProvider>
);

describe("usePlanDetailStore", () => {
  test("phase slice toggles active phases", () => {
    const { result } = renderHook(
      () => ({
        active: usePlanDetailStore((s) => s.activePhases),
        toggle: usePlanDetailStore((s) => s.togglePhase),
      }),
      { wrapper }
    );

    expect(result.current.active.has("changes")).toBe(true);
    act(() => result.current.toggle("changes"));
    expect(result.current.active.has("changes")).toBe(false);
  });

  test("editing slice tracks scopes and isEditing", () => {
    const { result } = renderHook(
      () => ({
        scopes: usePlanDetailStore((s) => s.editingScopes),
        setEditing: usePlanDetailStore((s) => s.setEditing),
      }),
      { wrapper }
    );

    expect(Object.keys(result.current.scopes)).toHaveLength(0);
    act(() => result.current.setEditing("title", true));
    expect(result.current.scopes.title).toBe(true);
    act(() => result.current.setEditing("title", false));
    expect(result.current.scopes.title).toBeUndefined();
  });

  test("each provider mount creates an isolated store", () => {
    const { result: a } = renderHook(
      () => usePlanDetailStore((s) => s.togglePhase),
      {
        wrapper,
      }
    );
    const { result: b } = renderHook(
      () => usePlanDetailStore((s) => s.togglePhase),
      {
        wrapper,
      }
    );
    expect(a.current).not.toBe(b.current);
  });
});
