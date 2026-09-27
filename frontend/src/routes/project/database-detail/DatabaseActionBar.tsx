import {
  ChevronDown,
  ChevronRight,
  type LucideIcon,
  MoreHorizontal,
} from "lucide-react";
import { type ReactNode, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSubmenu,
  DropdownMenuSubmenuContent,
  DropdownMenuSubmenuTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export interface DatabaseAction {
  key: string;
  label: string;
  icon: LucideIcon;
  disabled?: boolean;
  onClick?: () => void;
  options?: { key: string | number; label: string; onClick: () => void }[];
  wrap: (content: ReactNode) => ReactNode;
}

export function DatabaseActionBar({
  actions,
  primary,
}: {
  // Actions are ordered by retention priority, highest first.
  actions: DatabaseAction[];
  primary: ReactNode;
}) {
  const { t } = useTranslation();
  const containerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLDivElement>(null);
  const primaryRef = useRef<HTMLDivElement>(null);
  const [visibleCount, setVisibleCount] = useState(actions.length);

  useLayoutEffect(() => {
    const container = containerRef.current;
    const measurement = measureRef.current;
    const primaryElement = primaryRef.current;
    if (!container || !measurement || !primaryElement) return;
    const update = () => {
      const available = container.getBoundingClientRect().width;
      if (!available) return;
      const gap = Number.parseFloat(getComputedStyle(container).columnGap) || 8;
      const widths = Array.from(
        measurement.children,
        (child) => child.getBoundingClientRect().width
      );
      const moreWidth = widths.pop() ?? 0;
      const primaryWidth = primaryElement.getBoundingClientRect().width;
      let count = widths.length;
      while (count > 0) {
        const hasOverflow = count < widths.length;
        const required =
          primaryWidth +
          widths.slice(0, count).reduce((sum, width) => sum + width, 0) +
          (hasOverflow ? moreWidth : 0) +
          gap * (count + Number(hasOverflow));
        if (required <= available) break;
        count--;
      }
      setVisibleCount(count);
    };
    update();
    const observer = new ResizeObserver(update);
    observer.observe(container);
    observer.observe(primaryElement);
    for (const child of measurement.children) observer.observe(child);
    return () => observer.disconnect();
  }, [actions]);

  const inline = actions.slice(0, visibleCount).reverse();
  const overflow = actions.slice(visibleCount);
  return (
    <div
      ref={containerRef}
      className="relative flex min-w-0 flex-1 flex-wrap items-center justify-end gap-2"
    >
      <div
        aria-hidden="true"
        inert
        className="pointer-events-none invisible absolute inset-0 overflow-hidden"
      >
        <div ref={measureRef} className="flex w-max gap-2">
          {actions.map(({ key, label, icon: Icon, options }) => (
            <span
              key={key}
              className={buttonVariants({ appearance: "outline" })}
            >
              <Icon className="size-4" />
              {label}
              {options && <ChevronDown className="size-4" />}
            </span>
          ))}
          <span className={buttonVariants({ appearance: "outline" })}>
            <MoreHorizontal className="size-4" />
            {t("common.more")}
          </span>
        </div>
      </div>
      {overflow.length > 0 && (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button appearance="outline" />}>
            <MoreHorizontal className="size-4" />
            {t("common.more")}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            {overflow.map((action) => (
              <div key={action.key}>
                {action.wrap(
                  action.options ? (
                    <DropdownMenuSubmenu>
                      <DropdownMenuSubmenuTrigger disabled={action.disabled}>
                        <action.icon className="size-4" />
                        {action.label}
                        <ChevronRight className="size-4" />
                      </DropdownMenuSubmenuTrigger>
                      <DropdownMenuSubmenuContent>
                        {action.options.map((option) => (
                          <DropdownMenuItem
                            key={option.key}
                            onClick={option.onClick}
                          >
                            {option.label}
                          </DropdownMenuItem>
                        ))}
                      </DropdownMenuSubmenuContent>
                    </DropdownMenuSubmenu>
                  ) : (
                    <DropdownMenuItem
                      disabled={action.disabled}
                      onClick={action.onClick}
                    >
                      <action.icon className="size-4" />
                      {action.label}
                    </DropdownMenuItem>
                  )
                )}
              </div>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
      {inline.map((action) => (
        <div key={action.key} className="shrink-0">
          {action.wrap(
            action.options ? (
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={<Button appearance="outline" />}
                  disabled={action.disabled}
                >
                  <action.icon className="size-4" />
                  {action.label}
                  <ChevronDown className="size-4" />
                </DropdownMenuTrigger>
                <DropdownMenuContent>
                  {action.options.map((option) => (
                    <DropdownMenuItem key={option.key} onClick={option.onClick}>
                      {option.label}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <Button
                appearance="outline"
                disabled={action.disabled}
                onClick={action.onClick}
              >
                <action.icon className="size-4" />
                {action.label}
              </Button>
            )
          )}
        </div>
      ))}
      <div ref={primaryRef} className="shrink-0">
        {primary}
      </div>
    </div>
  );
}
