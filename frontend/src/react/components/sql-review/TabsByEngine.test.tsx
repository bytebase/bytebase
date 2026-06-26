import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let TabsByEngine: typeof import("./TabsByEngine").TabsByEngine;

vi.mock("@/react/components/EngineIcon", () => ({
  EngineIcon: () => <span data-testid="engine-icon" />,
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () =>
      act(() => {
        root.render(element);
      }),
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const rule = (engine: Engine, type: SQLReviewRule_Type): RuleTemplateV2 => ({
  engine,
  type,
  category: "ENGINE",
  level: 1,
  componentList: [],
});

beforeEach(async () => {
  ({ TabsByEngine } = await import("./TabsByEngine"));
});

describe("TabsByEngine", () => {
  test("can defer rendering inactive engine panels", () => {
    const ruleMapByEngine = new Map([
      [
        Engine.MYSQL,
        new Map([
          [
            SQLReviewRule_Type.ENGINE_MYSQL_USE_INNODB,
            rule(Engine.MYSQL, SQLReviewRule_Type.ENGINE_MYSQL_USE_INNODB),
          ],
        ]),
      ],
      [
        Engine.POSTGRES,
        new Map([
          [
            SQLReviewRule_Type.STATEMENT_DISALLOW_COMMIT,
            rule(Engine.POSTGRES, SQLReviewRule_Type.STATEMENT_DISALLOW_COMMIT),
          ],
        ]),
      ],
    ]);
    const renderPanel = vi.fn((_: RuleTemplateV2[], engine: Engine) => (
      <div data-testid={`engine-panel-${engine}`} />
    ));

    const { container, render, unmount } = renderIntoContainer(
      <TabsByEngine ruleMapByEngine={ruleMapByEngine} lazyPanels>
        {renderPanel}
      </TabsByEngine>
    );

    render();

    expect(
      container.querySelectorAll("[data-testid^='engine-panel-']")
    ).toHaveLength(1);
    expect(
      container.querySelector(`[data-testid='engine-panel-${Engine.MYSQL}']`)
    ).toBeTruthy();
    expect(
      container.querySelector(`[data-testid='engine-panel-${Engine.POSTGRES}']`)
    ).toBeNull();

    unmount();
  });
});
