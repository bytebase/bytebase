import type { AnchorHTMLAttributes, MouseEvent, ReactNode } from "react";
import { type RouteTarget, router } from "@/react/router";

export type RouterLinkProps = Omit<
  AnchorHTMLAttributes<HTMLAnchorElement>,
  "href"
> & {
  to: RouteTarget;
  children?: ReactNode;
};

export function RouterLink({
  to,
  children,
  onClick,
  target,
  download,
  ...props
}: RouterLinkProps) {
  const href = router.resolve(to).href;

  const handleClick = (event: MouseEvent<HTMLAnchorElement>) => {
    onClick?.(event);
    if (
      event.defaultPrevented ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey ||
      event.button !== 0 ||
      (target && target !== "_self") ||
      download != null
    ) {
      return;
    }

    event.preventDefault();
    router.push(to);
  };

  return (
    <a
      {...props}
      href={href}
      target={target}
      download={download}
      onClick={handleClick}
    >
      {children}
    </a>
  );
}
