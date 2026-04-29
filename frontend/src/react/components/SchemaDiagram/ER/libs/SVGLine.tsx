import { curveMonotoneX, line as d3Line } from "d3-shape";
import { type ComponentProps, useMemo, useState } from "react";
import { cn } from "@/react/lib/utils";
import { calcBBox } from "../../common/geometry";
import type { Path, Point, Rect } from "../../types";

const GLOW_WIDTH = 12;
const PADDING = GLOW_WIDTH / 2;
const COLORS = {
  GLOW: {
    NORMAL: "transparent",
    HOVER: "rgba(55,48,163,0.1)",
  },
  LINE: {
    NORMAL: "#1f2937",
    HOVER: "#4f46e5",
  },
};

interface SVGLineProps {
  path: Path;
  decorators?: Path[];
  className?: string;
}

/**
 * React port of `frontend/src/components/SchemaDiagram/ER/libs/SVGLine.vue`.
 *
 * Renders a polyline + optional decorator paths inside an absolutely
 * positioned SVG. Hover (across both the visible line and a wider
 * invisible "glow" track) flips the stroke color and width.
 */
export function SVGLine({
  path,
  decorators = [],
  className,
  ...rest
}: SVGLineProps & Omit<ComponentProps<"svg">, "viewBox" | "style" | "path">) {
  const [trackHover, setTrackHover] = useState(false);
  const [lineHover, setLineHover] = useState(false);
  const hover = trackHover || lineHover;

  const bbox = useMemo(() => {
    const points: Point[] = [...path];
    for (const d of decorators) {
      for (const p of d) {
        points.push(p);
      }
    }
    return calcBBox(points);
  }, [path, decorators]);

  const viewBox: Rect = useMemo(
    () => ({
      x: -PADDING,
      y: -PADDING,
      width: Math.max(bbox.width, 0) + PADDING * 2,
      height: Math.max(bbox.height, 0) + PADDING * 2,
    }),
    [bbox]
  );

  const normalize = (x: number, y: number): [number, number] => [
    x - bbox.x,
    y - bbox.y,
  ];

  const svgLine = useMemo(() => {
    const points: [number, number][] = path.map((p) => normalize(p.x, p.y));
    return d3Line<[number, number]>().curve(curveMonotoneX)(points) ?? "";
  }, [path, bbox]);

  const svgDecorators = useMemo(() => {
    return decorators.map((d) => {
      const points: [number, number][] = d.map((p) => normalize(p.x, p.y));
      return d3Line<[number, number]>()(points) ?? "";
    });
  }, [decorators, bbox]);

  return (
    <svg
      version="1.1"
      xmlns="http://www.w3.org/2000/svg"
      // The hover state lifts the line above its peers so the highlight
      // isn't occluded by an adjacent FK line. Component-internal stacking
      // within the same canvas — `z-10` is well below the layering
      // scanner's high-z threshold (40) and avoids the inline-zIndex ban.
      className={cn(
        "absolute cursor-pointer",
        hover ? "z-10" : "z-0",
        className
      )}
      pointerEvents="none"
      viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.width} ${viewBox.height}`}
      style={{
        left: `${bbox.x + viewBox.x}px`,
        top: `${bbox.y + viewBox.y}px`,
        width: `${viewBox.width}px`,
        height: `${viewBox.height}px`,
      }}
      {...rest}
    >
      <path
        d={svgLine}
        pointerEvents="visibleStroke"
        fill="none"
        stroke={hover ? COLORS.GLOW.HOVER : COLORS.GLOW.NORMAL}
        strokeWidth={GLOW_WIDTH}
        onMouseEnter={() => setTrackHover(true)}
        onMouseLeave={() => setTrackHover(false)}
      />
      {svgDecorators.map((d, i) => (
        <path
          key={i}
          d={d}
          fill="none"
          stroke={hover ? COLORS.LINE.HOVER : COLORS.LINE.NORMAL}
          strokeWidth={1}
          pointerEvents="none"
        />
      ))}
      <path
        d={svgLine}
        pointerEvents="visibleStroke"
        fill="none"
        stroke={hover ? COLORS.LINE.HOVER : COLORS.LINE.NORMAL}
        strokeWidth={hover ? 2 : 1.5}
        onMouseEnter={() => setLineHover(true)}
        onMouseLeave={() => setLineHover(false)}
      />
    </svg>
  );
}
