import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { defineComponent, h, nextTick } from "vue";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("@/store", () => ({
  pinia: { install: vi.fn() },
}));
vi.mock("@/plugins/i18n", () => ({
  default: { install: vi.fn() },
}));
vi.mock("@/plugins/naive-ui", () => ({
  default: { install: vi.fn() },
}));

import { VueMount } from "./VueMount";

const renderIntoContainer = (element: React.ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  return {
    container,
    render: () =>
      act(() => {
        root.render(element);
      }),
    rerender: (next: React.ReactElement) =>
      act(() => {
        root.render(next);
      }),
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

const Greet = defineComponent({
  props: { name: { type: String, default: "world" } },
  setup(props) {
    return () => h("span", { class: "greet" }, `hello ${props.name}`);
  },
});

describe("VueMount", () => {
  test("mounts a Vue component into the React container", () => {
    const { container, render, unmount } = renderIntoContainer(
      <VueMount component={Greet} props={{ name: "Stage 21" }} />
    );
    render();
    const span = container.querySelector("span.greet");
    expect(span?.textContent).toBe("hello Stage 21");
    unmount();
  });

  test("propagates prop changes without remounting", async () => {
    const { container, render, rerender, unmount } = renderIntoContainer(
      <VueMount component={Greet} props={{ name: "first" }} />
    );
    render();
    const before = container.querySelector("span.greet");
    expect(before?.textContent).toBe("hello first");

    rerender(<VueMount component={Greet} props={{ name: "second" }} />);
    await nextTick();
    const after = container.querySelector("span.greet");
    expect(after?.textContent).toBe("hello second");
    expect(after).toBe(before); // same DOM node — Vue updated in place.
    unmount();
  });

  test("unmounts the Vue app when React unmounts", () => {
    const teardown = vi.fn();
    const Probe = defineComponent({
      setup() {
        return () => h("span", { class: "probe" }, "alive");
      },
      beforeUnmount: teardown,
    });

    const { container, render, unmount } = renderIntoContainer(
      <VueMount component={Probe} />
    );
    render();
    expect(container.querySelector("span.probe")?.textContent).toBe("alive");

    unmount();
    expect(teardown).toHaveBeenCalledTimes(1);
  });
});
