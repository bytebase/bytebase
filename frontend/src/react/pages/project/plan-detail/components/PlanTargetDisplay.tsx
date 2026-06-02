import { ChevronRight } from "lucide-react";
import { useMemo } from "react";
import { EngineIcon } from "@/react/components/EngineIcon";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { isValidDatabaseName } from "@/types";
import { unknownDatabase } from "@/types/v1/database";
import { extractDatabaseResourceName, getInstanceResource } from "@/utils";

type PlanTargetDisplaySize = "sm" | "md";

const sizeClasses: Record<
  PlanTargetDisplaySize,
  {
    database: string;
    environment: string;
    icon: string;
    instance: string;
    root: string;
    separator: string;
  }
> = {
  sm: {
    database: "min-w-12",
    environment: "max-w-24",
    icon: "h-4 w-4",
    instance: "max-w-40",
    root: "text-sm",
    separator: "h-4 w-4",
  },
  md: {
    database: "min-w-16",
    environment: "max-w-28",
    icon: "h-5 w-5",
    instance: "max-w-48",
    root: "text-base",
    separator: "h-5 w-5",
  },
};

export function PlanTargetDisplay({
  className,
  showEngine = true,
  showEnvironment = false,
  showInstance = true,
  size = "sm",
  target,
}: {
  className?: string;
  showEngine?: boolean;
  showEnvironment?: boolean;
  showInstance?: boolean;
  size?: PlanTargetDisplaySize;
  target: string;
}) {
  const databasesByName = useAppStore((s) => s.databasesByName);
  const environmentList = useAppStore((s) => s.environmentList);
  const classes = sizeClasses[size];

  const database = databasesByName[target] ?? unknownDatabase();
  const environmentName =
    database.effectiveEnvironment ??
    database.instanceResource?.environment ??
    "";
  const environment = useMemo(
    () =>
      environmentName
        ? useAppStore.getState().getEnvironmentByName(environmentName)
        : undefined,
    [environmentList, environmentName]
  );

  if (!isValidDatabaseName(target)) {
    return (
      <span
        className={cn(
          "truncate text-control-placeholder",
          classes.root,
          className
        )}
      >
        {target}
      </span>
    );
  }

  const instance = getInstanceResource(database);
  const { databaseName } = extractDatabaseResourceName(target);
  const shouldShowSeparator = showInstance && Boolean(instance.title);
  const title = [
    showEnvironment ? environment?.title : undefined,
    showInstance ? instance.title : undefined,
    databaseName,
  ]
    .filter(Boolean)
    .join(" / ");

  return (
    <div
      className={cn(
        "inline-flex max-w-full min-w-0 items-center",
        classes.root,
        className
      )}
      title={title}
    >
      {showEngine && (
        <EngineIcon
          engine={instance.engine}
          className={cn("mr-1 shrink-0", classes.icon)}
        />
      )}
      {showEnvironment && environment?.title && (
        <span
          className={cn(
            "mr-1 min-w-0 shrink truncate text-control-placeholder",
            classes.environment
          )}
        >
          {environment.title}
        </span>
      )}
      {showInstance && instance.title && (
        <span
          className={cn(
            "min-w-0 shrink-[2] truncate text-control-light",
            classes.instance
          )}
        >
          {instance.title}
        </span>
      )}
      {shouldShowSeparator && (
        <ChevronRight
          className={cn("shrink-0 text-control-light/80", classes.separator)}
        />
      )}
      <span className={cn("flex-1 truncate text-control", classes.database)}>
        {databaseName}
      </span>
    </div>
  );
}
