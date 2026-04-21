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
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/components/ui/layer", () => ({
  getLayerRoot: mocks.getLayerRoot,
  LAYER_SURFACE_CLASS: "layer-surface",
}));

vi.mock("@/react/components/ui/combobox-position", () => ({
  getPortalDropdownStyle: vi.fn(() => ({ top: 100, left: 0, width: 200 })),
  isPortalDropdownStyleEqual: vi.fn(() => true),
  shouldIgnorePortalDropdownScroll: vi.fn(() => false),
}));

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: ({
    value,
    onChange,
  }: {
    value: string;
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  }) => <input data-testid="search" value={value} onChange={onChange} />,
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
});
