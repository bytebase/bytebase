import { act, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { SshConnectionForm } from "./SshConnectionForm";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

type SshValue = {
  sshHost: string;
  sshPort: string;
  sshUser: string;
  sshPassword: string;
  sshPrivateKey: string;
};

const emptySshValue = (): SshValue => ({
  sshHost: "",
  sshPort: "",
  sshUser: "",
  sshPassword: "",
  sshPrivateKey: "",
});

function ControlledSshConnectionForm() {
  const [value, setValue] = useState<SshValue>(emptySshValue);
  return (
    <SshConnectionForm
      value={value}
      onChange={(partial) => setValue((prev) => ({ ...prev, ...partial }))}
    />
  );
}

function mount(node: React.ReactNode) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(node);
  });
  return { container, root };
}

function setInputValue(input: HTMLInputElement, value: string) {
  const setter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    "value"
  )?.set;
  setter?.call(input, value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
}

describe("SshConnectionForm", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders an SSH tunnel switch", () => {
    const { container, root } = mount(<ControlledSshConnectionForm />);
    expect(container.querySelector('[role="switch"]')).toHaveAttribute(
      "aria-checked",
      "false"
    );

    act(() => {
      root.unmount();
    });
  });

  test("keeps tunnel selected after editing SSH fields before port is entered", () => {
    const { container, root } = mount(<ControlledSshConnectionForm />);
    const tunnelSwitch = container.querySelector(
      '[role="switch"]'
    ) as HTMLElement | null;
    expect(tunnelSwitch).not.toBeNull();

    act(() => {
      tunnelSwitch?.click();
    });

    let userInput = container.querySelector(
      "#sshUser"
    ) as HTMLInputElement | null;
    expect(userInput).not.toBeNull();

    act(() => {
      setInputValue(userInput!, "alice");
    });

    userInput = container.querySelector("#sshUser");
    expect(userInput).not.toBeNull();
    expect(container.querySelector('[role="switch"]')).toHaveAttribute(
      "aria-checked",
      "true"
    );

    act(() => {
      root.unmount();
    });
  });
  test("a focus request waits for the tunnel fields, then focuses the host", () => {
    function Requested({ focusRequest }: { focusRequest: number }) {
      const [value, setValue] = useState<SshValue>(emptySshValue);
      return (
        <>
          <button
            type="button"
            data-testid="enable"
            onClick={() => setValue((prev) => ({ ...prev, sshPort: "22" }))}
          />
          <SshConnectionForm
            value={value}
            focusRequest={focusRequest}
            onChange={(partial) => setValue((prev) => ({ ...prev, ...partial }))}
          />
        </>
      );
    }
    const scrollIntoView = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoView;
    const { container, root } = mount(<Requested focusRequest={0} />);

    // The request arrives while the tunnel is still off.
    act(() => {
      root.render(<Requested focusRequest={1} />);
    });
    expect(container.querySelector("#sshHost")).toBeNull();

    vi.useFakeTimers();
    try {
      act(() => {
        (
          container.querySelector("[data-testid='enable']") as HTMLButtonElement
        ).click();
      });
      // The fields have just mounted and the layout may still replace them.
      expect(document.activeElement).not.toBe(
        container.querySelector("#sshHost")
      );

      act(() => {
        vi.advanceTimersByTime(50);
      });
      expect(document.activeElement).toBe(container.querySelector("#sshHost"));
      expect(scrollIntoView).toHaveBeenCalledOnce();
    } finally {
      vi.useRealTimers();
    }

    act(() => {
      root.unmount();
    });
  });
});
