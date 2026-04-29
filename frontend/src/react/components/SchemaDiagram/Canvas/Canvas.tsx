import { Image, LayoutGrid } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { ZOOM_RANGE } from "../common/const";
import { useSchemaDiagramContext } from "../common/context";
import { useDragCanvas, useFitView, useSetCenter } from "./composables";
import { useScreenshot } from "./composables/useScreenshot";
import { ZoomButton } from "./ZoomButton";

interface CanvasProps {
  /** Diagram content (TableNodes, FK lines) — placed inside the
   *  transformed `desktop` layer. */
  children?: ReactNode;
  /** Extra control buttons rendered to the left of the screenshot /
   *  fit / zoom group. */
  controls?: ReactNode;
  /**
   * When provided, the screenshot button is shown and clicking it runs
   * `html-to-image` on the canvas DOM, downloading + copying the result.
   * Omit to hide the button entirely.
   */
  screenshotFilename?: () => string;
}

/**
 * React port of `Canvas/Canvas.vue`. The viewport: applies a CSS
 * `transform: matrix(zoom, 0, 0, zoom, x, y)` to a desktop layer and
 * mounts the zoom + fit + screenshot button group at the bottom-right.
 */
export function Canvas({
  children,
  controls,
  screenshotFilename,
}: CanvasProps) {
  const { t } = useTranslation();
  const ctx = useSchemaDiagramContext();
  const { zoom, position } = ctx;

  const [canvas, setCanvas] = useState<HTMLDivElement | null>(null);
  const handleFitView = useFitView(canvas);
  const { handleZoom } = useDragCanvas(canvas);
  useSetCenter(canvas);
  const { capture, capturing } = useScreenshot(canvas, {
    filename: screenshotFilename ?? (() => "schema-diagram.png"),
  });

  return (
    <div
      ref={setCanvas}
      className="w-full h-full relative bg-control-bg overflow-hidden"
    >
      <div
        className="absolute overflow-visible"
        style={{
          transformOrigin: "0 0 0",
          transform: `matrix(${zoom}, 0, 0, ${zoom}, ${position.x}, ${position.y})`,
        }}
      >
        {children}
      </div>

      <div
        className="absolute right-2 bottom-2 flex items-center gap-x-2"
        data-screenshot-hide
      >
        {controls}

        {screenshotFilename && (
          <Tooltip content="Screenshot" side="top">
            <Button
              variant="outline"
              size="xs"
              onClick={() => void capture()}
              disabled={capturing}
              aria-label="Screenshot"
              className="bg-background"
            >
              <Image className="size-3" />
            </Button>
          </Tooltip>
        )}

        <Tooltip content={t("schema-diagram.fit-content-with-view")} side="top">
          <Button
            variant="outline"
            size="xs"
            onClick={handleFitView}
            aria-label={t("schema-diagram.fit-content-with-view")}
            className="bg-background"
          >
            <LayoutGrid className="size-3" />
          </Button>
        </Tooltip>

        <ZoomButton
          min={ZOOM_RANGE.min}
          max={ZOOM_RANGE.max}
          onZoomIn={() => handleZoom(0.1)}
          onZoomOut={() => handleZoom(-0.1)}
        />
      </div>
    </div>
  );
}
