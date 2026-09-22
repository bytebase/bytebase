import { type RefObject, useEffect, useRef, useState } from "react";

// Whether an element has ever come near the viewport, for work a page should
// defer until the reader gets there. Latches true and stops observing; an
// environment without IntersectionObserver reports true right away.
export function useInViewOnce<T extends Element>(
  rootMargin = "200px"
): { ref: RefObject<T | null>; inView: boolean } {
  const ref = useRef<T>(null);
  const [inView, setInView] = useState(
    () => typeof IntersectionObserver === "undefined"
  );
  useEffect(() => {
    const element = ref.current;
    if (inView || !element) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries.some((entry) => entry.isIntersecting)) return;
        setInView(true);
        observer.disconnect();
      },
      { rootMargin }
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [inView, rootMargin]);
  return { ref, inView };
}
