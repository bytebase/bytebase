import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  currentUser: {
    name: "users/alice@example.com",
    email: "alice@example.com",
    mfaEnabled: false,
  },
  saas: false,
  requestReauthCode: vi.fn(async () => ({})),
  pushNotification: vi.fn(),
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => mocks.currentUser,
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/stores/app", () => {
  const getState = () => ({ serverInfo: { saas: mocks.saas } });
  return {
    useAppStore: Object.assign(
      (selector?: (state: ReturnType<typeof getState>) => unknown) =>
        selector ? selector(getState()) : getState(),
      { getState }
    ),
  };
});

vi.mock("@/api", () => ({
  userServiceClientConnect: { requestReauthCode: mocks.requestReauthCode },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

let CredentialProofInput: typeof import(
  "./CredentialProofInput"
).CredentialProofInput;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
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

// The proof input is the only control in a re-authentication dialog, so a
// screen reader that cannot name it leaves the account with no way to tell
// which secret is being asked for.
const expectLabelledInput = (container: HTMLElement, labelKey: string) => {
  const label = container.querySelector("label");
  expect(label?.textContent).toBe(`${labelKey}*`);
  expect(label?.htmlFor).toBeTruthy();
  const input = container.querySelector<HTMLInputElement>(
    `#${CSS.escape(label?.htmlFor ?? "")}`
  );
  expect(input?.tagName).toBe("INPUT");
  return input as HTMLInputElement;
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.currentUser = {
    name: "users/alice@example.com",
    email: "alice@example.com",
    mfaEnabled: false,
  };
  mocks.saas = false;
  ({ CredentialProofInput } = await import("./CredentialProofInput"));
});

describe("CredentialProofInput", () => {
  test("password mode labels its input", () => {
    const { container, render, unmount } = renderIntoContainer(
      <CredentialProofInput value="" onChange={() => {}} />
    );
    render();

    const input = expectLabelledInput(
      container,
      "credential-proof.password-label"
    );
    expect(input.type).toBe("password");
    expect(input.autocomplete).toBe("current-password");

    unmount();
  });

  test("factor mode labels its input", () => {
    mocks.currentUser = { ...mocks.currentUser, mfaEnabled: true };
    const { container, render, unmount } = renderIntoContainer(
      <CredentialProofInput value="" onChange={() => {}} />
    );
    render();

    const input = expectLabelledInput(
      container,
      "credential-proof.factor-label"
    );
    expect(input.placeholder).toBe("credential-proof.factor-placeholder");
    expect(input.autocomplete).toBe("one-time-code");

    unmount();
  });

  test("email mode labels the input it pairs with the send button", () => {
    mocks.saas = true;
    const { container, render, unmount } = renderIntoContainer(
      <CredentialProofInput value="" onChange={() => {}} />
    );
    render();

    const input = expectLabelledInput(container, "credential-proof.email-label");
    expect(input.placeholder).toBe("credential-proof.email-placeholder");
    // The button shares the field's row, so the label must point at the input
    // rather than wrapping both controls.
    const button = container.querySelector("button");
    expect(button?.textContent).toBe("credential-proof.email-me-a-code");

    unmount();
  });
});
