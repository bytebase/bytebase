import { useCallback, useState } from "react";
import { pushNotification } from "@/store";
import { useSchemaDiagramContext } from "../../common/context";
import { calcBBox } from "../../common/geometry";

const HIDE_FLAG_ATTR = "data-bb-screenshot-active";
const CAPTURE_PADDING = 40;

interface UseScreenshotOptions {
  /** Callback that resolves the desired filename (e.g. `${db}.png`). */
  filename: () => string;
}

interface UseScreenshotTargets {
  /** The outer canvas (clipped viewport). Used only as a fallback. */
  canvas: Element | null;
  /**
   * The inner `desktop` layer. `overflow-visible`, holds the diagram at
   * natural coordinates with its own transform. We capture this so
   * tables/FK lines outside the visible viewport are still included.
   */
  desktop: Element | null;
}

/**
 * Capture the SchemaDiagram canvas as a PNG. Targets the inner `desktop`
 * layer (not the outer viewport — that's `overflow-hidden` and would
 * clip large schemas to whatever fits on screen). Computes the natural
 * bbox from the geometries the diagram has registered with the context,
 * then overrides the cloned node's transform + dimensions so
 * `html-to-image` captures the full extent at zoom 1.
 *
 * Returns:
 *   - `capture()` — async, opens the download + copies to clipboard
 *   - `capturing` — boolean, true while the capture is in flight
 */
export function useScreenshot(
  targets: UseScreenshotTargets,
  { filename }: UseScreenshotOptions
) {
  const ctx = useSchemaDiagramContext();
  const { setBusy, geometries } = ctx;
  const [capturing, setCapturing] = useState(false);

  const capture = useCallback(async () => {
    const { canvas, desktop } = targets;
    const target = desktop ?? canvas;
    if (!target || capturing) return;
    setCapturing(true);
    setBusy(true);
    // Mark the document so the CSS rule that hides `data-screenshot-hide`
    // affordances (zoom buttons, schema selector, etc.) takes effect on
    // the cloned subtree.
    document.body.setAttribute(HIDE_FLAG_ATTR, "true");

    try {
      const [{ toBlob }, { default: download }] = await Promise.all([
        import("html-to-image"),
        import("downloadjs"),
      ]);

      // Compute the natural bbox from registered geometries (TableNodes
      // call `useGeometry(rect)` and ForeignKeyLines call
      // `useGeometry(path)`). Add padding so edges aren't flush against
      // the PNG border.
      const bbox = calcBBox([...geometries]);
      const captureWidth = Math.max(
        Math.ceil(bbox.width + bbox.x + CAPTURE_PADDING * 2),
        100
      );
      const captureHeight = Math.max(
        Math.ceil(bbox.height + bbox.y + CAPTURE_PADDING * 2),
        100
      );

      const blob = await toBlob(target as HTMLElement, {
        pixelRatio: 1,
        quality: 0.9,
        width: captureWidth,
        height: captureHeight,
        // The cloned root style — overrides the live transform/size so
        // the capture renders at zoom 1 with origin shifted to (padding,
        // padding). Children (TableNodes, FK lines) keep their absolute
        // positions in natural coordinates and are visible because the
        // desktop layer is `overflow-visible`.
        style: {
          transform: `translate(${CAPTURE_PADDING}px, ${CAPTURE_PADDING}px) scale(1)`,
          transformOrigin: "0 0",
          width: `${captureWidth}px`,
          height: `${captureHeight}px`,
          backgroundColor: "white",
        },
      });
      if (blob) {
        download(blob, filename(), blob.type);
        try {
          const data = [new window.ClipboardItem({ [blob.type]: blob })];
          await navigator.clipboard.write(data);
        } catch {
          // ClipboardItem may be unavailable (Firefox < 127 etc.) — the
          // download itself still succeeded, so swallow this branch.
        }
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title:
            "Screenshot generated successfully and copied to the clipboard!",
        });
      }
    } catch (err) {
      console.error("Screenshot failed", err);
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Screenshot failed",
        description: String(err),
      });
    } finally {
      document.body.removeAttribute(HIDE_FLAG_ATTR);
      setCapturing(false);
      setBusy(false);
    }
  }, [targets, capturing, filename, setBusy, geometries]);

  return { capture, capturing };
}
