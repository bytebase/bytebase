import type { ComponentPropsWithoutRef } from "react";
import { EngineIconPath } from "@/react/components/instance/constants";
import { cn } from "@/react/lib/utils";
import type { Engine } from "@/types/proto-es/v1/common_pb";

type EngineIconProps = Omit<ComponentPropsWithoutRef<"img">, "src"> & {
  engine: Engine;
};

export function EngineIcon({
  engine,
  alt = "",
  className,
  ...props
}: EngineIconProps) {
  const src = getEngineIconSrc(engine);
  if (!src) {
    return null;
  }

  return (
    <img
      src={src}
      alt={alt}
      className={cn("shrink-0 object-contain", className)}
      {...props}
    />
  );
}

export function getEngineIconSrc(engine: Engine) {
  return EngineIconPath[engine];
}
