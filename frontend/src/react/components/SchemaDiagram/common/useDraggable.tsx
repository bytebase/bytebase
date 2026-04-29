import { useEffect, useRef } from "react";

export type DraggableOptions = {
  exact?: boolean;
  onStart?: () => void;
  onPan?: (dx: number, dy: number) => void;
  onEnd?: () => void;
  capture?: boolean;
};

/**
 * React port of `frontend/src/components/SchemaDiagram/common/useDraggable.ts`.
 *
 * Pointer-driven drag listener. Attaches `pointerdown` to the given
 * element and `pointermove` / `pointerup` to `window`, mirroring the
 * Vue `vueuse` setup so a drag continues even if the pointer leaves
 * the element. `screenX`/`screenY` are used (matching Vue) so the
 * delta is independent of the element's position when it's the one
 * being dragged.
 *
 * Pass `null` while the element is unmounted; the hook re-attaches
 * when a real element arrives. Callbacks are read through a ref so
 * inline option objects don't re-run the effect each render.
 */
export const useDraggable = (
  target: Element | null,
  options: DraggableOptions
) => {
  const optionsRef = useRef(options);
  optionsRef.current = options;

  useEffect(() => {
    if (!target) return;

    let startPointer: { x: number; y: number } | undefined;
    let lastPointer: { x: number; y: number } | undefined;

    const start = (e: PointerEvent) => {
      const opts = optionsRef.current;
      if (opts.exact && e.target !== target) return;
      startPointer = { x: e.screenX, y: e.screenY };
      lastPointer = { x: e.screenX, y: e.screenY };
      e.stopPropagation();
      e.preventDefault();
      opts.onStart?.();
    };
    const move = (e: PointerEvent) => {
      if (!startPointer || !lastPointer) return;
      const opts = optionsRef.current;
      const pointer = { x: e.screenX, y: e.screenY };
      e.stopPropagation();
      e.preventDefault();
      opts.onPan?.(pointer.x - lastPointer.x, pointer.y - lastPointer.y);
      lastPointer = pointer;
    };
    const end = (e: PointerEvent) => {
      const opts = optionsRef.current;
      startPointer = undefined;
      lastPointer = undefined;
      e.stopPropagation();
      e.preventDefault();
      opts.onEnd?.();
    };

    const capture = optionsRef.current.capture ?? false;
    target.addEventListener("pointerdown", start as EventListener, capture);
    window.addEventListener("pointermove", move, capture);
    window.addEventListener("pointerup", end, capture);
    return () => {
      target.removeEventListener(
        "pointerdown",
        start as EventListener,
        capture
      );
      window.removeEventListener("pointermove", move, capture);
      window.removeEventListener("pointerup", end, capture);
    };
  }, [target]);
};
