import { act, useEffect, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import {
  type EnvironmentSelectionItem,
  getEnvironmentListKey,
  resolveSelectedEnvironmentId,
} from "./environmentSelection";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const createEnvironment = (id: string): EnvironmentSelectionItem => ({
  id,
});

function Harness({
  environments,
  hash,
}: {
  environments: EnvironmentSelectionItem[];
  hash: string;
}) {
  const [selectedId, setSelectedId] = useState("");
  const environmentListKey = getEnvironmentListKey(environments);

  useEffect(() => {
    setSelectedId((currentId) =>
      resolveSelectedEnvironmentId({
        currentId,
        environmentList: environments,
        hash,
      })
    );
  }, [environmentListKey, hash]);

  return <div data-selected-id={selectedId}>{selectedId}</div>;
}

describe("environmentSelection", () => {
  let container: HTMLDivElement | undefined;
  let root: Root | undefined;

  afterEach(() => {
    if (root) {
      act(() => {
        root?.unmount();
      });
    }
    container?.remove();
    container = undefined;
    root = undefined;
  });

  test("returns the same key for equivalent environment ids", () => {
    expect(getEnvironmentListKey([createEnvironment("prod")])).toEqual(
      getEnvironmentListKey([createEnvironment("prod")])
    );
    expect(
      getEnvironmentListKey([
        createEnvironment("prod"),
        createEnvironment("dev"),
      ])
    ).not.toEqual(
      getEnvironmentListKey([
        createEnvironment("dev"),
        createEnvironment("prod"),
      ])
    );
  });

  test("keeps the current selection when the hash is missing", () => {
    expect(
      resolveSelectedEnvironmentId({
        currentId: "prod",
        environmentList: [createEnvironment("dev"), createEnvironment("prod")],
        hash: "",
      })
    ).toBe("prod");
  });

  test("prefers a valid hash and falls back to the first environment", () => {
    expect(
      resolveSelectedEnvironmentId({
        currentId: "",
        environmentList: [createEnvironment("dev"), createEnvironment("prod")],
        hash: "prod",
      })
    ).toBe("prod");

    expect(
      resolveSelectedEnvironmentId({
        currentId: "missing",
        environmentList: [createEnvironment("dev"), createEnvironment("prod")],
        hash: "unknown",
      })
    ).toBe("dev");
  });

  test("stabilizes selection when the list identity changes without id changes", () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    expect(() => {
      act(() => {
        root?.render(
          <Harness environments={[createEnvironment("prod")]} hash="" />
        );
      });
      act(() => {
        root?.render(
          <Harness environments={[createEnvironment("prod")]} hash="" />
        );
      });
    }).not.toThrow();

    expect(container.textContent).toBe("prod");
  });
});
