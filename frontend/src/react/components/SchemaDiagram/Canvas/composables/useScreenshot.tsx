import { useCallback, useState } from "react";
import { pushNotification } from "@/store";
import { useSchemaDiagramContext } from "../../common/context";

const HIDE_FLAG_ATTR = "data-bb-screenshot-active";

interface UseScreenshotOptions {
  /** Callback that resolves the desired filename (e.g. `${db}.png`). */
  filename: () => string;
}

/**
 * React port of the Vue `DummyCanvas.capture()` flow. Instead of
 * re-mounting the diagram off-screen (Vue's approach with
 * `teleport to="#capture-container"`), we run `html-to-image` on the
 * live canvas DOM after temporarily flagging chrome affordances with
 * `data-screenshot-hide` to keep them out of the export.
 *
 * Returns:
 *   - `capture()` — async, opens the download + copies to clipboard
 *   - `capturing` — boolean, true while the capture is in flight
 *
 * Usage:
 *   ```tsx
 *   const [canvas, setCanvas] = useState<HTMLDivElement | null>(null);
 *   const { capture, capturing } = useScreenshot(canvas, {
 *     filename: () => `${dbName}.png`,
 *   });
 *   ```
 */
export function useScreenshot(
  canvas: Element | null,
  { filename }: UseScreenshotOptions
) {
  const ctx = useSchemaDiagramContext();
  const { setBusy } = ctx;
  const [capturing, setCapturing] = useState(false);

  const capture = useCallback(async () => {
    if (!canvas || capturing) return;
    setCapturing(true);
    setBusy(true);
    // Mark the document so the CSS rule that hides `data-screenshot-hide`
    // affordances (zoom buttons, schema selector, etc.) takes effect.
    document.body.setAttribute(HIDE_FLAG_ATTR, "true");

    try {
      const [{ toBlob }, { default: download }] = await Promise.all([
        import("html-to-image"),
        import("downloadjs"),
      ]);
      const blob = await toBlob(canvas as HTMLElement, {
        pixelRatio: 1,
        quality: 0.9,
      });
      if (blob) {
        download(blob, filename(), blob.type);
        try {
          const data = [new window.ClipboardItem({ [blob.type]: blob })];
          await navigator.clipboard.write(data);
        } catch {
          // ClipboardItem may be unavailable (Firefox older than 127, etc.) —
          // the download itself still succeeded, so swallow the clipboard
          // failure rather than surfacing an error.
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
  }, [canvas, capturing, filename, setBusy]);

  return { capture, capturing };
}
