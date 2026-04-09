import { mount } from "@vue/test-utils";
import { NDataTable } from "naive-ui";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { defineComponent, h } from "vue";
import Drawer from "@/components/v2/Container/Drawer.vue";
import { LegacyVueBridgeProviders } from "./mountLegacyVueApp";

vi.mock("@/plugins/i18n", () => ({
  default: {
    install: vi.fn(),
  },
  locale: {
    value: "en-US",
  },
}));

vi.mock("@/plugins/naive-ui", () => ({
  default: {
    install: vi.fn(),
  },
}));

vi.mock("@/router", () => ({
  router: {
    install: vi.fn(),
  },
}));

vi.mock("@/store", () => ({
  pinia: {
    install: vi.fn(),
  },
}));

const Probe = defineComponent({
  name: "LegacyBridgeProbe",
  render() {
    return h("div", [
      h(NDataTable, {
        columns: [{ key: "name", title: "Name" }],
        data: [{ name: "orders" }],
      }),
      h(Drawer, { show: false }),
    ]);
  },
});

describe("LegacyVueBridgeProviders", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  test("raw legacy mounts are missing the app provider shell", () => {
    const warnings: string[] = [];
    vi.spyOn(console, "warn").mockImplementation((...args) => {
      warnings.push(args.join(" "));
    });

    expect(() => mount(Probe)).toThrow();
    expect(
      warnings.some((warning) =>
        warning.includes("Symbol(bb.overlay-stack-manager)")
      )
    ).toBe(true);
  });

  test("provides the Naive UI and overlay contexts expected by legacy mounts", () => {
    const warnings: string[] = [];
    vi.spyOn(console, "warn").mockImplementation((...args) => {
      warnings.push(args.join(" "));
    });

    expect(() =>
      mount(LegacyVueBridgeProviders, {
        slots: {
          default: () => h(Probe),
        },
      })
    ).not.toThrow();

    expect(
      warnings.some((warning) => warning.includes("n-config-provider"))
    ).toBe(false);
    expect(
      warnings.some((warning) =>
        warning.includes("Symbol(bb.overlay-stack-manager)")
      )
    ).toBe(false);
  });
});
