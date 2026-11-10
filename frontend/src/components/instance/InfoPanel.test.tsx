import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  InfoSection,
  InfoSnippetContentKey,
  InfoSnippetLinkTitleKey,
} from "./info-content";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/types", () => ({
  DATASOURCE_ADMIN_USER_NAME: "bytebase",
}));

// CopyButton pulls in the app store (and its connect/i18n chain), which isn't
// available in this minimal render harness. The i18n assertions don't touch it.
vi.mock("@/components/ui/copy-button", () => ({
  CopyButton: () => null,
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const infoSnippetCases: Array<
  [Engine, InfoSection, InfoSnippetContentKey, InfoSnippetLinkTitleKey]
> = [
  [
    Engine.POSTGRES,
    "host",
    "instance.info.postgresql.host.content",
    "instance.info.connect-instance.link",
  ],
  [
    Engine.POSTGRES,
    "authentication",
    "instance.info.postgresql.authentication.content",
    "instance.info.configure-database-user.link",
  ],
  [
    Engine.POSTGRES,
    "ssl",
    "instance.info.postgresql.ssl.content",
    "instance.info.ssl-tls-connection.link",
  ],
  [
    Engine.POSTGRES,
    "ssh",
    "instance.info.postgresql.ssh.content",
    "instance.info.ssh-tunnel.link",
  ],
  [
    Engine.MYSQL,
    "host",
    "instance.info.mysql.host.content",
    "instance.info.connect-instance.link",
  ],
  [
    Engine.MYSQL,
    "authentication",
    "instance.info.mysql.authentication.content",
    "instance.info.configure-database-user.link",
  ],
  [
    Engine.MYSQL,
    "ssl",
    "instance.info.mysql.ssl.content",
    "instance.info.ssl-tls-connection.link",
  ],
  [
    Engine.MYSQL,
    "ssh",
    "instance.info.mysql.ssh.content",
    "instance.info.ssh-tunnel.link",
  ],
  [
    Engine.MONGODB,
    "host",
    "instance.info.mongodb.host.content",
    "instance.info.connect-instance.link",
  ],
  [
    Engine.MONGODB,
    "authentication",
    "instance.info.mongodb.authentication.content",
    "instance.info.configure-database-user.link",
  ],
  [
    Engine.MONGODB,
    "ssl",
    "instance.info.mongodb.ssl.content",
    "instance.info.ssl-tls-connection.link",
  ],
  [
    Engine.MONGODB,
    "ssh",
    "instance.info.mongodb.ssh.content",
    "instance.info.ssh-tunnel.link",
  ],
];

describe("InfoPanel i18n", () => {
  test.each(
    infoSnippetCases
  )("renders %s %s guidance from locale keys", async (engine, section, contentKey, linkTitleKey) => {
    const { InfoPanelContent } = await import("./InfoPanel");
    const { container, render, unmount } = renderIntoContainer(
      <InfoPanelContent engine={engine} section={section} />
    );

    render();

    expect(container.querySelector("p")?.textContent).toBe(contentKey);
    expect(container.querySelector("a.accent-link")?.textContent).toBe(
      linkTitleKey
    );

    unmount();
  });

  test.each([
    Engine.MYSQL,
    Engine.TIDB,
    Engine.MARIADB,
    Engine.OCEANBASE,
  ])("grants no privilege that only some versions accept for engine %s", async (engine) => {
    const { InfoPanelContent } = await import("./InfoPanel");
    const { container, render, unmount } = renderIntoContainer(
      <InfoPanelContent engine={engine} section="authentication" />
    );

    render();

    const code = container.querySelector("code")?.textContent ?? "";
    expect(code).toContain("ON *.* to bytebase@'%';");
    expect(code).not.toMatch(
      /\/\*!|SET_USER_ID|SET_ANY_DEFINER|ALLOW_NONEXISTENT_DEFINER/
    );

    unmount();
  });

  test("warns about the MySQL definer privilege and links to the MySQL docs", async () => {
    const { InfoPanelContent } = await import("./InfoPanel");
    const { container, render, unmount } = renderIntoContainer(
      <InfoPanelContent engine={Engine.MYSQL} section="authentication" />
    );

    render();

    const alert = container.querySelector("[role=alert]");
    expect(alert?.textContent).toContain(
      "instance.sentence.create-user-example.mysql.warn"
    );
    expect(alert?.querySelector("a")?.getAttribute("href")).toBe(
      "https://docs.bytebase.com/get-started/connect/mysql?source=console#create-a-user-for-bytebase"
    );

    unmount();
  });

  test.each([
    Engine.TIDB,
    Engine.MARIADB,
    Engine.OCEANBASE,
  ])("shows no MySQL definer warning for engine %s", async (engine) => {
    const { InfoPanelContent } = await import("./InfoPanel");
    const { container, render, unmount } = renderIntoContainer(
      <InfoPanelContent engine={engine} section="authentication" />
    );

    render();

    expect(container.querySelector("code")).not.toBeNull();
    expect(container.querySelector("[role=alert]")).toBeNull();

    unmount();
  });

  test("renders sync database guidance from locale keys", async () => {
    const { InfoPanelContent } = await import("./InfoPanel");
    const { container, render, unmount } = renderIntoContainer(
      <InfoPanelContent engine={Engine.POSTGRES} section="sync-databases" />
    );

    render();

    expect(container.querySelector("p")?.textContent).toBe(
      "instance.info.sync-databases.content"
    );

    unmount();
  });
});
