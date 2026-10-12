import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  ConnectionRecovery,
  normalizeConnectionFailureCategory,
} from "./ConnectionRecovery";

const mocks = vi.hoisted(() => ({
  isSaaSMode: false,
  egressIpAddresses: [] as string[],
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (
    selector: (state: {
      isSaaSMode: () => boolean;
      serverInfo: { egressIpAddresses: string[] };
    }) => unknown
  ) =>
    selector({
      isSaaSMode: () => mocks.isSaaSMode,
      serverInfo: { egressIpAddresses: mocks.egressIpAddresses },
    }),
}));

vi.mock("@/components/ui/copy-button", () => ({
  CopyButton: ({ content }: { content: string }) =>
    createElement("button", { "data-copy": content }),
}));

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("ConnectionRecovery", () => {
  beforeEach(() => {
    mocks.isSaaSMode = false;
    mocks.egressIpAddresses = [];
  });

  const renderRecovery = async (
    props: Parameters<typeof ConnectionRecovery>[0]
  ) => {
    const container = document.createElement("div");
    const root = createRoot(container);
    await act(async () => {
      root.render(createElement(ConnectionRecovery, props));
    });
    return { container, unmount: () => act(() => root.unmount()) };
  };

  test("normalizes backend connection failure categories", () => {
    expect(normalizeConnectionFailureCategory("auth_failed")).toBe(
      "auth_failed"
    );
    expect(normalizeConnectionFailureCategory("ssl_tls_failed")).toBe(
      "ssl_tls_failed"
    );
    expect(normalizeConnectionFailureCategory("not_a_category")).toBe(
      "unknown"
    );
    expect(normalizeConnectionFailureCategory(undefined)).toBe("unknown");
  });

  test("shows Cloud-specific network recovery in SaaS mode", async () => {
    mocks.isSaaSMode = true;
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        createElement(ConnectionRecovery, { category: "network_unreachable" })
      );
    });

    expect(container.textContent).toContain(
      "instance.connection-recovery.network.description-saas"
    );
    expect(container.textContent).toContain(
      "instance.connection-recovery.network.steps.firewall-saas"
    );
    expect(container.querySelector("a")).toHaveAttribute(
      "href",
      "https://docs.bytebase.com/get-started/cloud#prerequisites"
    );

    act(() => root.unmount());
  });

  test("shows deployment-specific network recovery when self-hosted", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        createElement(ConnectionRecovery, { category: "network_unreachable" })
      );
    });

    expect(container.textContent).toContain(
      "instance.connection-recovery.network.description-self-hosted"
    );
    expect(container.textContent).toContain(
      "instance.connection-recovery.network.steps.firewall-self-hosted"
    );
    expect(container.querySelector("a")).toHaveAttribute(
      "href",
      "https://docs.bytebase.com/get-started/connect/overview?source=console"
    );

    act(() => root.unmount());
  });
  test("recognizes the unreachable-host categories", () => {
    expect(normalizeConnectionFailureCategory("host_not_found")).toBe(
      "host_not_found"
    );
    expect(normalizeConnectionFailureCategory("private_address")).toBe(
      "private_address"
    );
  });

  test.each(["network_unreachable", "timeout"] as const)(
    "shows the Cloud addresses to allow for %s",
    async (category) => {
      mocks.isSaaSMode = true;
      mocks.egressIpAddresses = ["203.0.113.7", "198.51.100.0/24"];
      const view = await renderRecovery({ category });

      expect(view.container.textContent).toContain(
        "instance.sentence.firewall-info-addresses"
      );
      expect(view.container.textContent).toContain("203.0.113.7");
      expect(
        [...view.container.querySelectorAll("[data-copy]")].map((button) =>
          button.getAttribute("data-copy")
        )
      ).toEqual(["203.0.113.7", "198.51.100.0/24"]);
      expect(view.container.textContent).not.toContain(
        "instance.connection-recovery.network.steps.firewall-saas"
      );
      view.unmount();
    }
  );

  test("never shows the Cloud addresses on a self-hosted deployment", async () => {
    mocks.egressIpAddresses = ["203.0.113.7"];
    const view = await renderRecovery({ category: "network_unreachable" });
    expect(view.container.textContent).not.toContain("203.0.113.7");
    view.unmount();
  });

  test("names the private host that Cloud cannot reach", async () => {
    mocks.isSaaSMode = true;
    const view = await renderRecovery({
      category: "private_address",
      host: "10.0.0.5",
    });
    expect(view.container.textContent).toContain(
      "instance.connection-recovery.private.title"
    );
    expect(view.container.querySelector("ul")).toBeNull();
    view.unmount();
  });

  test("explains an unresolved hostname per deployment", async () => {
    const selfHosted = await renderRecovery({ category: "host_not_found" });
    expect(selfHosted.container.textContent).toContain(
      "instance.connection-recovery.host-not-found.description-self-hosted"
    );
    selfHosted.unmount();

    mocks.isSaaSMode = true;
    const cloud = await renderRecovery({ category: "host_not_found" });
    expect(cloud.container.textContent).toContain(
      "instance.connection-recovery.host-not-found.description-saas"
    );
    cloud.unmount();
  });

  test("renders each action as a button with what it gives up", async () => {
    const onClick = vi.fn();
    const view = await renderRecovery({
      category: "ssl_tls_failed",
      actions: [
        {
          key: "retry",
          label: "Retry unverified",
          note: "Not verified",
          onClick,
        },
        { key: "busy", label: "Busy", disabled: true, onClick: vi.fn() },
      ],
    });

    const buttons = [...view.container.querySelectorAll("button")];
    expect(buttons.map((button) => button.textContent)).toEqual([
      "Retry unverified",
      "Busy",
    ]);
    expect(buttons[1]).toBeDisabled();
    expect(view.container.textContent).toContain("Not verified");

    act(() => buttons[0].click());
    expect(onClick).toHaveBeenCalledOnce();
    view.unmount();
  });
});
