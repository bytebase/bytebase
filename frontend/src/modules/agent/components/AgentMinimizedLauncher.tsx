import { Bot } from "lucide-react";
import { useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { WINDOW_MARGIN } from "../window";

export function AgentMinimizedLauncher({ onOpen }: { onOpen: () => void }) {
  const { t } = useTranslation();
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [bottom, setBottom] = useState(WINDOW_MARGIN);

  useLayoutEffect(() => {
    const button = buttonRef.current;
    if (!button) return;
    let footers: Element[] = [];
    const updatePosition = () => {
      const { width, height } = button.getBoundingClientRect();
      const viewportHeight = window.innerHeight;
      const right = window.innerWidth - WINDOW_MARGIN;
      const left = right - width;
      const maximumBottom = Math.max(
        WINDOW_MARGIN,
        viewportHeight - WINDOW_MARGIN - height
      );
      let nextBottom = WINDOW_MARGIN;
      const footerRects = footers
        .filter((footer) => {
          const style = getComputedStyle(footer);
          return style.display !== "none" && style.visibility !== "hidden";
        })
        .map((footer) => footer.getBoundingClientRect())
        .filter((rect) => rect.width > 0 && rect.height > 0)
        .sort((a, b) => b.top - a.top);

      // Measure collisions from the resting position, not the displaced button.
      for (const rect of footerRects) {
        const buttonBottom = viewportHeight - nextBottom;
        if (
          rect.left < right &&
          rect.right > left &&
          rect.top < buttonBottom &&
          rect.bottom > buttonBottom - height
        ) {
          nextBottom = Math.min(
            maximumBottom,
            viewportHeight - rect.top + WINDOW_MARGIN
          );
        }
      }
      setBottom((current) => (current === nextBottom ? current : nextBottom));
    };
    const resizeObserver = new ResizeObserver(updatePosition);
    const syncFooters = () => {
      footers = [
        ...document.querySelectorAll('[data-slot="sticky-action-footer"]'),
      ];
      resizeObserver.disconnect();
      resizeObserver.observe(button);
      for (const footer of footers) resizeObserver.observe(footer);
      updatePosition();
    };
    const mutationObserver = new MutationObserver(syncFooters);
    mutationObserver.observe(document.body, { childList: true, subtree: true });
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    syncFooters();
    return () => {
      resizeObserver.disconnect();
      mutationObserver.disconnect();
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, []);

  return (
    <Button
      ref={buttonRef}
      type="button"
      size="lg"
      data-agent-window
      aria-label={t("agent.self")}
      title={t("agent.self")}
      className="fixed right-4 w-10 rounded-full px-0 shadow-lg"
      style={{ bottom }}
      onClick={onOpen}
    >
      <Bot className="size-5" aria-hidden="true" />
    </Button>
  );
}
