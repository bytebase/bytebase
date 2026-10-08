import { act, cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { useInViewOnce } from "./useInViewOnce";

type Callback = (entries: { isIntersecting: boolean }[]) => void;
class FakeObserver {
  static instances: FakeObserver[] = [];
  observe = vi.fn();
  disconnect = vi.fn();
  constructor(readonly callback: Callback) {
    FakeObserver.instances.push(this);
  }
}

function Probe() {
  const { ref, inView } = useInViewOnce<HTMLDivElement>();
  return <div data-inview={inView} data-testid="probe" ref={ref} />;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  FakeObserver.instances = [];
});

describe("useInViewOnce", () => {
  test("latches true the first time the element intersects, then stops observing", () => {
    vi.stubGlobal("IntersectionObserver", FakeObserver);
    const { getByTestId } = render(<Probe />);
    expect(getByTestId("probe").getAttribute("data-inview")).toBe("false");
    const [observer] = FakeObserver.instances;
    expect(observer.observe).toHaveBeenCalledWith(getByTestId("probe"));
    act(() => observer.callback([{ isIntersecting: false }]));
    expect(getByTestId("probe").getAttribute("data-inview")).toBe("false");
    act(() => observer.callback([{ isIntersecting: true }]));
    expect(getByTestId("probe").getAttribute("data-inview")).toBe("true");
    expect(observer.disconnect).toHaveBeenCalled();
    expect(FakeObserver.instances).toHaveLength(1);
  });

  test("reports in view right away without IntersectionObserver", () => {
    vi.stubGlobal("IntersectionObserver", undefined);
    const { getByTestId } = render(<Probe />);
    expect(getByTestId("probe").getAttribute("data-inview")).toBe("true");
  });
});
