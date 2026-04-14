import { afterEach, describe, expect, test } from "vitest";
import {
  getLayerRoot,
  LAYER_ROOT_ID,
  LAYER_Z_INDEX,
} from "./layer";

describe("layer roots", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("creates each layer root once with stable ordering", () => {
    const overlay = getLayerRoot("overlay");
    const agent = getLayerRoot("agent");
    const critical = getLayerRoot("critical");

    expect(overlay.id).toBe(LAYER_ROOT_ID.overlay);
    expect(agent.id).toBe(LAYER_ROOT_ID.agent);
    expect(critical.id).toBe(LAYER_ROOT_ID.critical);

    expect(overlay.style.zIndex).toBe(String(LAYER_Z_INDEX.overlay));
    expect(agent.style.zIndex).toBe(String(LAYER_Z_INDEX.agent));
    expect(critical.style.zIndex).toBe(String(LAYER_Z_INDEX.critical));

    expect(document.body.children[0]?.id).toBe(LAYER_ROOT_ID.overlay);
    expect(document.body.children[1]?.id).toBe(LAYER_ROOT_ID.agent);
    expect(document.body.children[2]?.id).toBe(LAYER_ROOT_ID.critical);

    expect(getLayerRoot("agent")).toBe(agent);
  });
});
