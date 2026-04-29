import { Minus, Plus } from "lucide-react";
import { useMemo } from "react";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import { useSchemaDiagramContext } from "../common/context";

interface ZoomButtonProps {
  min: number;
  max: number;
  onZoomIn: () => void;
  onZoomOut: () => void;
}

/**
 * React port of `Canvas/ZoomButton.vue`. A three-button group: minus,
 * current zoom (read-only label), plus.
 */
export function ZoomButton({ min, max, onZoomIn, onZoomOut }: ZoomButtonProps) {
  const { zoom } = useSchemaDiagramContext();
  const display = useMemo(() => `${Math.round(zoom * 100)}%`, [zoom]);

  return (
    <div className="bg-background rounded-sm flex items-center gap-x-px">
      <Button
        variant="outline"
        size="xs"
        onClick={onZoomOut}
        disabled={zoom <= min}
        className={cn("rounded-r-none px-2")}
        aria-label="Zoom out"
      >
        <Minus className="size-3" />
      </Button>
      <div className="px-2 h-6 inline-flex items-center justify-center text-xs border-y border-control-border bg-background">
        <span className="w-8 text-center">{display}</span>
      </div>
      <Button
        variant="outline"
        size="xs"
        onClick={onZoomIn}
        disabled={zoom >= max}
        className={cn("rounded-l-none px-2")}
        aria-label="Zoom in"
      >
        <Plus className="size-3" />
      </Button>
    </div>
  );
}
