import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  getLayerRoot: vi.fn(() => document.body),
  getPortalDropdownStyle: vi.fn(() => ({ top: 100, left: 0, width: 200 })),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/components/ui/layer", () => ({
  getLayerRoot: mocks.getLayerRoot,
  LAYER_SURFACE_CLASS: "layer-surface",
}));

vi.mock("@/components/ui/combobox-position", () => ({
  getPortalDropdownStyle: mocks.getPortalDropdownStyle,
  isPortalDropdownStyleEqual: vi.fn(() => true),
  shouldIgnorePortalDropdownScroll: vi.fn(() => false),
}));

vi.mock("@/components/ui/search-input", () => ({
  SearchInput: ({
    value,
    onChange,
  }: {
    value: string;
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  }) => <input data-testid="search" value={value} onChange={onChange} />,
}));

vi.mock("@/components/HighlightLabelText", () => ({
  HighlightLabelText: ({
    keyword,
    text,
  }: {
    keyword?: string;
    text: string;
  }) => <b data-keyword={keyword}>{text}</b>,
}));

let ConnectChooser: typeof import("./ConnectChooser").ConnectChooser;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

const defaultOptions = [
  { value: "-1", label: "Unspecified" },
  { value: "public", label: "public" },
];

beforeEach(async () => {
  vi.clearAllMocks();
  ({ ConnectChooser } = await import("./ConnectChooser"));
});

describe("ConnectChooser", () => {
  test("renders placeholder text when isChosen=false", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectChooser
        value=""
        onChange={vi.fn()}
        options={defaultOptions}
        isChosen={false}
        placeholder="Select schema"
      />
    );
    render();
    expect(container.textContent).toContain("Select schema");
    unmount();
  });

  test("renders truncated value when isChosen=true", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectChooser
        value="public"
        onChange={vi.fn()}
        options={defaultOptions}
        isChosen={true}
        placeholder="Select schema"
      />
    );
    render();
    expect(container.textContent).toContain("public");
    unmount();
  });

  test("applies custom dropdown sizing class without widening the trigger", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectChooser
        value="ED"
        onChange={vi.fn()}
        options={defaultOptions}
        isChosen={true}
        placeholder="Select container"
        dropdownClassName="min-w-[12rem]"
      />
    );
    render();

    expect(container.querySelector("button")?.className).not.toContain(
      "min-w-[12rem]"
    );
    act(() => {
      container.querySelector("button")?.click();
    });
    expect(
      document.body.querySelector(".min-w-\\[12rem\\]")
    ).not.toBeNull();
    unmount();
  });

  test("passes custom dropdown minimum width to portal positioning", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectChooser
        value="ED"
        onChange={vi.fn()}
        options={defaultOptions}
        isChosen={true}
        placeholder="Select container"
        dropdownMinWidth={192}
      />
    );
    render();

    act(() => {
      container.querySelector("button")?.click();
    });
    expect(mocks.getPortalDropdownStyle).toHaveBeenCalledWith(
      expect.any(Object),
      expect.any(Number),
      expect.any(Number),
      {
        minWidth: 192,
        viewportWidth: expect.any(Number),
      }
    );
    unmount();
  });

  test("does not render a chevron in the run trigger", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectChooser
        value="-1"
        onChange={vi.fn()}
        options={defaultOptions}
        isChosen={false}
        placeholder="Select container"
        triggerVariant="run"
      />
    );
    render();

    expect(container.querySelector("button")?.querySelectorAll("svg")).toHaveLength(
      1
    );
    unmount();
  });

  test("renders db.schema.default when isChosen=true but value is empty string", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectChooser
        value=""
        onChange={vi.fn()}
        options={defaultOptions}
        isChosen={true}
        placeholder="Select schema"
      />
    );
    render();
    // t() returns the key when mocked, so we get "db.schema.default"
    expect(container.textContent).toContain("db.schema.default");
    unmount();
  });

  test("Network icon is present in trigger", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectChooser
        value=""
        onChange={vi.fn()}
        options={defaultOptions}
        isChosen={false}
        placeholder="Select schema"
      />
    );
    render();
    const button = container.querySelector("button");
    expect(button).not.toBeNull();
    // Network icon renders as an svg
    expect(button?.querySelector("svg")).not.toBeNull();
    unmount();
  });

  test("onChange fires when an option is selected", () => {
    const onChange = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <ConnectChooser
        value=""
        onChange={onChange}
        options={defaultOptions}
        isChosen={false}
        placeholder="Select schema"
      />
    );
    render();
    // Open the dropdown
    act(() => {
      container.querySelector("button")?.click();
    });
    // Find option buttons in the portal (document.body)
    const optionButtons = document.body.querySelectorAll(
      "button:not([aria-label])"
    );
    // Click first option
    act(() => {
      (optionButtons[0] as HTMLButtonElement)?.click();
    });
    expect(onChange).toHaveBeenCalled();
    unmount();
  });

  test("highlights matching option text", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ConnectChooser
        value=""
        onChange={vi.fn()}
        options={defaultOptions}
        isChosen={false}
        placeholder="Select schema"
      />
    );
    render();
    act(() => container.querySelector("button")?.click());

    const input = document.body.querySelector(
      '[data-testid="search"]'
    ) as HTMLInputElement;
    act(() => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set;
      valueSetter?.call(input, "pub");
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });

    const highlight = Array.from(document.body.querySelectorAll("b")).find(
      (element) => element.textContent === "public"
    );
    expect(highlight?.dataset.keyword).toBe("pub");
    unmount();
  });

  test("opens the dropdown when the external open signal changes", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    document.body.appendChild(container);
    const render = (openSignal: number) => {
      act(() => {
        root.render(
          <ConnectChooser
            value=""
            onChange={vi.fn()}
            options={defaultOptions}
            isChosen={false}
            placeholder="Select schema"
            openSignal={openSignal}
          />
        );
      });
    };

    render(0);
    expect(document.body.textContent).not.toContain("Unspecified");

    render(1);

    expect(document.body.textContent).toContain("Unspecified");
    act(() => {
      root.unmount();
    });
    container.remove();
  });
});
