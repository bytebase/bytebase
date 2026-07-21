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

  test("keeps tunnel selected after editing SSH fields before port is entered", () => {
    const { container, root } = mount(<ControlledSshConnectionForm />);
    const tunnelRadio = container.querySelector(
      'input[value="TUNNEL+PK"]'
    ) as HTMLInputElement | null;
    expect(tunnelRadio).not.toBeNull();

    act(() => {
      tunnelRadio?.click();
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
    expect(
      (container.querySelector('input[value="TUNNEL+PK"]') as HTMLInputElement)
        .checked
    ).toBe(true);

    act(() => {
      root.unmount();
    });
  });
});
