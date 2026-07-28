import { afterEach, describe, expect, test } from "vitest";
import { getLayerRoot, LAYER_ROOT_ID, LAYER_Z_INDEX } from "./layer";

describe("layer roots", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("creates each layer root once with stable ordering", () => {
    const critical = getLayerRoot("critical");
    const watermark = getLayerRoot("watermark");
    const overlay = getLayerRoot("overlay");
    const agent = getLayerRoot("agent");

    expect(overlay.id).toBe(LAYER_ROOT_ID.overlay);
    expect(agent.id).toBe(LAYER_ROOT_ID.agent);
    expect(critical.id).toBe(LAYER_ROOT_ID.critical);
    expect(watermark.id).toBe(LAYER_ROOT_ID.watermark);

    expect(overlay.style.zIndex).toBe(String(LAYER_Z_INDEX.overlay));
    expect(agent.style.zIndex).toBe(String(LAYER_Z_INDEX.agent));
    expect(critical.style.zIndex).toBe(String(LAYER_Z_INDEX.critical));
    expect(watermark.style.zIndex).toBe(String(LAYER_Z_INDEX.watermark));

    expect(document.body.children[0]?.id).toBe(LAYER_ROOT_ID.overlay);
    expect(document.body.children[1]?.id).toBe(LAYER_ROOT_ID.agent);
    expect(document.body.children[2]?.id).toBe(LAYER_ROOT_ID.critical);
    expect(document.body.children[3]?.id).toBe(LAYER_ROOT_ID.watermark);

    expect(getLayerRoot("overlay")).toBe(overlay);
    expect(getLayerRoot("agent")).toBe(agent);
    expect(getLayerRoot("critical")).toBe(critical);
    expect(getLayerRoot("watermark")).toBe(watermark);

    expect(document.body.children).toHaveLength(4);
    expect(document.querySelectorAll(`#${LAYER_ROOT_ID.overlay}`)).toHaveLength(
      1
    );
    expect(document.querySelectorAll(`#${LAYER_ROOT_ID.agent}`)).toHaveLength(
      1
    );
    expect(
      document.querySelectorAll(`#${LAYER_ROOT_ID.critical}`)
    ).toHaveLength(1);
    expect(
      document.querySelectorAll(`#${LAYER_ROOT_ID.watermark}`)
    ).toHaveLength(1);
  });

  test("keeps the critical layer above the legacy Naive overlay stack", () => {
    expect(LAYER_Z_INDEX.critical).toBeGreaterThan(6000);
  });

  test("keeps the watermark above blocking app surfaces", () => {
    expect(LAYER_Z_INDEX.watermark).toBeGreaterThan(LAYER_Z_INDEX.critical);
  });
});
