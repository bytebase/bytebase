import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import i18n from "@/lib/i18n";
import { useSemanticTypes } from "./useSemanticTypes";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

const mocks = vi.hoisted(() => ({
  setting: undefined as unknown,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (
    selector: (state: { getSettingByName: () => unknown }) => unknown
  ) => selector({ getSettingByName: () => mocks.setting }),
}));

function SemanticTypeList() {
  const { configuredSemanticTypes, semanticTypes } = useSemanticTypes();
  return (
    <div
      data-configured={configuredSemanticTypes.map(({ id }) => id).join(",")}
      data-expanded={semanticTypes.map(({ id }) => id).join(",")}
      data-first-title={semanticTypes[0]?.title}
    />
  );
}

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  container = document.createElement("div");
  root = createRoot(container);
});

afterEach(async () => {
  act(() => root.unmount());
  await i18n.changeLanguage("en-US");
});

describe("useSemanticTypes", () => {
  test("puts built-ins before configured types without duplicates", () => {
    mocks.setting = {
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [
              { id: "email", title: "Email" },
              { id: "bb.default", title: "Stored default" },
            ],
          },
        },
      },
    };

    act(() => root.render(<SemanticTypeList />));

    const result = container.firstElementChild;
    expect(result?.getAttribute("data-configured")).toBe(
      "email,bb.default"
    );
    expect(result?.getAttribute("data-expanded")).toBe(
      "bb.default,bb.default-partial,email"
    );
  });

  test("updates when the cached setting changes", () => {
    mocks.setting = undefined;
    act(() => root.render(<SemanticTypeList />));
    expect(container.firstElementChild?.getAttribute("data-expanded")).toBe(
      "bb.default,bb.default-partial"
    );

    mocks.setting = {
      value: {
        value: {
          case: "semanticType",
          value: { types: [{ id: "phone", title: "Phone" }] },
        },
      },
    };
    act(() => root.render(<SemanticTypeList />));

    expect(container.firstElementChild?.getAttribute("data-expanded")).toBe(
      "bb.default,bb.default-partial,phone"
    );
  });

  test("updates built-in labels when the locale changes", async () => {
    await act(async () => {
      await i18n.changeLanguage("en-US");
      root.render(<SemanticTypeList />);
    });
    expect(
      container.firstElementChild?.getAttribute("data-first-title")
    ).toBe("Default");

    await act(async () => {
      await i18n.changeLanguage("zh-CN");
    });
    expect(
      container.firstElementChild?.getAttribute("data-first-title")
    ).toBe("全遮掩");
  });
});
