import type { MouseEventHandler, ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import type { Environment } from "@/types";
import { EnvironmentBadge } from "./EnvironmentLabel";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({
    children,
    className,
    onClick,
    to,
  }: {
    children: ReactNode;
    className?: string;
    onClick?: MouseEventHandler<HTMLAnchorElement>;
    to: { path?: string };
  }) => (
    <a className={className} data-path={to.path} onClick={onClick}>
      {children}
    </a>
  ),
}));

const render = async (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  await act(async () => {
    root.render(element);
    await Promise.resolve();
  });

  return { container, root };
};

const environment = {
  id: "prod",
  name: "environments/prod",
  title: "Production",
  color: "#16a34a",
  order: 0,
  tags: {},
} as Environment;

describe("EnvironmentBadge", () => {
  let root: Root | undefined;

  afterEach(() => {
    act(() => root?.unmount());
    document.body.innerHTML = "";
    root = undefined;
  });

  test("wraps the badge with RouterLink when link is enabled", async () => {
    const rendered = await render(
      <EnvironmentBadge
        environment={environment}
        hasEnvTierFeature={false}
        link
      />
    );
    root = rendered.root;

    const link = rendered.container.querySelector("a");
    expect(link?.getAttribute("data-path")).toBe("/environments/prod");
    expect(link?.className).toBe("hover:underline");
    expect(link?.textContent).toBe("Production");
  });
});
