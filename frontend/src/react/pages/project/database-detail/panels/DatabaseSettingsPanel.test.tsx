import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  updateDatabase: vi.fn(),
  pushNotification: vi.fn(),
  hasProjectPermissionV2: vi.fn(() => true),
  getDatabaseProject: vi.fn((database: { project: string }) => ({
    name: database.project,
  })),
  getInstanceResource: vi.fn(
    (database: { instanceResource?: { environment?: string } }) =>
      database.instanceResource ?? { environment: "" }
  ),
  EnvironmentSelect: vi.fn(
    ({
      value,
      onChange,
      disabled,
      className,
      clearable,
      renderSuffix,
    }: {
      value: string;
      onChange: (value: string) => void;
      disabled?: boolean;
      className?: string;
      clearable?: boolean;
      renderSuffix?: (environment: { name: string }) => React.ReactNode;
    }) => (
      <div data-testid="environment-select-wrapper" className={className}>
        <select
          data-testid="environment-select"
          value={value}
          disabled={disabled}
          onChange={(event) => onChange(event.target.value)}
        >
          <option value="">none</option>
          <option value="environments/dev">Dev</option>
          <option value="environments/test">Test</option>
        </select>
        <div data-testid="environment-option-environments/dev">
          Dev {renderSuffix?.({ name: "environments/dev" })}
        </div>
        <div data-testid="environment-option-environments/test">
          Test {renderSuffix?.({ name: "environments/test" })}
        </div>
        {clearable && (
          <button
            type="button"
            data-testid="environment-clear"
            onClick={() => onChange("")}
          >
            clear
          </button>
        )}
      </div>
    )
  ),
}));

let DatabaseSettingsPanel: typeof import("./DatabaseSettingsPanel").DatabaseSettingsPanel;

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/components/EnvironmentSelect", () => ({
  EnvironmentSelect: mocks.EnvironmentSelect,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: (props: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
  useDatabaseV1Store: () => ({
    updateDatabase: mocks.updateDatabase,
  }),
}));

vi.mock("@/utils", () => ({
  convertLabelsToKVList: (labels: Record<string, string>, sort = true) => {
    const list = Object.keys(labels).map((key) => ({
      key,
      value: labels[key],
    }));
    return sort ? list.sort((a, b) => a.key.localeCompare(b.key)) : list;
  },
  convertKVListToLabels: (
    list: { key: string; value: string }[],
    omitEmpty = true
  ) => {
    return list.reduce<Record<string, string>>((labels, kv) => {
      if (!kv.value && omitEmpty) {
        return labels;
      }
      labels[kv.key] = kv.value;
      return labels;
    }, {});
  },
  MAX_LABEL_VALUE_LENGTH: 63,
  getDatabaseProject: mocks.getDatabaseProject,
  getInstanceResource: mocks.getInstanceResource,
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: (nextElement = element) => {
      act(() => {
        root.render(nextElement);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

const setInputValue = (input: HTMLInputElement, value: string) => {
  act(() => {
    const descriptor = Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    );
    descriptor?.set?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
    input.dispatchEvent(new Event("change", { bubbles: true }));
  });
};

const changeSelectValue = (select: HTMLSelectElement, value: string) => {
  act(() => {
    const descriptor = Object.getOwnPropertyDescriptor(
      HTMLSelectElement.prototype,
      "value"
    );
    descriptor?.set?.call(select, value);
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });
};

const click = (element: HTMLElement) => {
  act(() => {
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
};

const makeDatabase = (
  overrides: Partial<{
    name: string;
    project: string;
    environment: string;
    effectiveEnvironment: string;
    labels: Record<string, string>;
    instanceResource: {
      environment?: string;
    };
  }> = {}
) =>
  ({
    name: "instances/inst1/databases/db1",
    project: "projects/proj1",
    environment: "",
    effectiveEnvironment: "environments/dev",
    labels: {},
    instanceResource: {
      environment: "",
    },
    ...overrides,
  }) as Database;

beforeEach(async () => {
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.updateDatabase.mockReset();
  mocks.updateDatabase.mockResolvedValue(undefined);
  mocks.pushNotification.mockReset();
  mocks.hasProjectPermissionV2.mockReset();
  mocks.hasProjectPermissionV2.mockReturnValue(true);
  mocks.getDatabaseProject.mockReset();
  mocks.getDatabaseProject.mockImplementation(
    (database: { project: string }) => ({
      name: database.project,
    })
  );
  mocks.getInstanceResource.mockReset();
  mocks.getInstanceResource.mockImplementation(
    (database: { instanceResource?: { environment?: string } }) =>
      database.instanceResource ?? { environment: "" }
  );
  mocks.EnvironmentSelect.mockClear();

  vi.resetModules();
  ({ DatabaseSettingsPanel } = await import("./DatabaseSettingsPanel"));
});

describe("DatabaseSettingsPanel", () => {
  test("updates the database environment", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseSettingsPanel, {
        database: makeDatabase(),
      })
    );

    render();

    const select = container.querySelector(
      '[data-testid="environment-select"]'
    ) as HTMLSelectElement | null;
    expect(select).not.toBeNull();

    changeSelectValue(select as HTMLSelectElement, "environments/test");
    await flush();

    expect(mocks.updateDatabase).toHaveBeenCalledWith(
      expect.objectContaining({
        database: expect.objectContaining({
          environment: "environments/test",
        }),
        updateMask: expect.objectContaining({
          paths: ["environment"],
        }),
      })
    );
    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "SUCCESS",
        title: "common.updated",
      })
    );

    unmount();
  });

  test("saves edited labels through the shared React label editor", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseSettingsPanel, {
        database: makeDatabase({
          labels: { team: "dba" },
        }),
      })
    );

    render();

    const valueInput = Array.from(container.querySelectorAll("input")).find(
      (input) => (input as HTMLInputElement).value === "dba"
    ) as HTMLInputElement | undefined;
    expect(valueInput).toBeDefined();

    setInputValue(valueInput as HTMLInputElement, "platform");
    await flush();

    const saveButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.save"
    ) as HTMLButtonElement | undefined;
    expect(saveButton).toBeDefined();

    click(saveButton as HTMLButtonElement);
    await flush();

    expect(mocks.updateDatabase).toHaveBeenCalledWith(
      expect.objectContaining({
        database: expect.objectContaining({
          labels: { team: "platform" },
        }),
        updateMask: expect.objectContaining({
          paths: ["labels"],
        }),
      })
    );
    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "SUCCESS",
        title: "common.updated",
      })
    );

    unmount();
  });

  test("blocks label save when the value exceeds the shared max length", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseSettingsPanel, {
        database: makeDatabase({
          labels: { team: "dba" },
        }),
      })
    );

    render();

    const valueInput = Array.from(container.querySelectorAll("input")).find(
      (input) => (input as HTMLInputElement).value === "dba"
    ) as HTMLInputElement | undefined;
    expect(valueInput).toBeDefined();

    setInputValue(valueInput as HTMLInputElement, "x".repeat(64));
    await flush();

    const saveButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.save"
    ) as HTMLButtonElement | undefined;
    expect(saveButton).toBeDefined();
    expect((saveButton as HTMLButtonElement).disabled).toBe(true);
    expect(container.textContent).toContain(
      "label.error.max-value-length-exceeded"
    );

    click(saveButton as HTMLButtonElement);
    await flush();

    expect(mocks.updateDatabase).not.toHaveBeenCalled();

    unmount();
  });

  test("clears the environment to an empty string when the instance has no default environment", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseSettingsPanel, {
        database: makeDatabase({
          effectiveEnvironment: "environments/dev",
          environment: "environments/dev",
          instanceResource: {
            environment: "",
          },
        }),
      })
    );

    render();

    const clearButton = container.querySelector(
      '[data-testid="environment-clear"]'
    ) as HTMLButtonElement | null;
    expect(clearButton).not.toBeNull();

    click(clearButton as HTMLButtonElement);
    await flush();

    expect(mocks.updateDatabase).toHaveBeenCalledWith(
      expect.objectContaining({
        database: expect.objectContaining({
          environment: "",
        }),
        updateMask: expect.objectContaining({
          paths: ["environment"],
        }),
      })
    );

    unmount();
  });

  test("marks the instance default environment inside the selector and disables clearing when inherited", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseSettingsPanel, {
        database: makeDatabase({
          effectiveEnvironment: "environments/test",
          environment: "environments/test",
          instanceResource: {
            environment: "environments/dev",
          },
        }),
      })
    );

    render();

    expect(container.textContent).toContain("common.default");
    expect(mocks.EnvironmentSelect).toHaveBeenCalledWith(
      expect.objectContaining({
        clearable: false,
        renderSuffix: expect.any(Function),
      }),
      undefined
    );
    expect(
      container.querySelector('[data-testid="environment-clear"]')
    ).toBeNull();

    unmount();
  });
});
