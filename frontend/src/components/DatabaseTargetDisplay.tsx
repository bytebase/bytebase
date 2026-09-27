import { useMemo } from "react";
import { EngineIcon } from "@/components/EngineIcon";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { HighlightLabelText } from "@/components/HighlightLabelText";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { isValidDatabaseName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils";

type DatabaseTargetDisplaySize = "sm" | "md";

const sizeClasses: Record<
  DatabaseTargetDisplaySize,
  {
    database: string;
    environment: string;
    icon: string;
    instance: string;
    root: string;
  }
> = {
  sm: {
    database: "min-w-12",
    environment: "max-w-24",
    icon: "h-4 w-4",
    instance: "max-w-40",
    root: "text-sm",
  },
  md: {
    database: "min-w-16",
    environment: "max-w-28",
    icon: "h-5 w-5",
    instance: "max-w-48",
    root: "text-base",
  },
};

type DatabaseTargetDisplayProps = {
  className?: string;
  showEngine?: boolean;
  showEnvironment?: boolean;
  showInstance?: boolean;
  size?: DatabaseTargetDisplaySize;
  keyword?: string | readonly string[];
} & (
  | {
      database?: Database;
      target: string;
    }
  | {
      database: Database;
      target?: string;
    }
);

export function DatabaseTargetDisplay({
  className,
  database: databaseProp,
  keyword,
  showEngine = true,
  showEnvironment = false,
  showInstance = true,
  size = "sm",
  target,
}: DatabaseTargetDisplayProps) {
  const databasesByName = useAppStore((s) => s.databasesByName);
  const environmentList = useAppStore((s) => s.environmentList);
  const classes = sizeClasses[size];

  const targetName = target ?? databaseProp?.name ?? "";
  const database = databaseProp ?? databasesByName[targetName];
  const instance = database?.instanceResource;
  const environmentName =
    database?.effectiveEnvironment ?? instance?.environment ?? "";
  const environment = useMemo(
    () =>
      environmentName
        ? useAppStore.getState().getEnvironmentByName(environmentName)
        : undefined,
    [environmentList, environmentName]
  );

  if (!isValidDatabaseName(targetName)) {
    return (
      <EllipsisText
        text={targetName}
        className={cn(
          "truncate text-control-placeholder",
          classes.root,
          className
        )}
      >
        <HighlightLabelText text={targetName} keyword={keyword} />
      </EllipsisText>
    );
  }

  const { databaseName, instanceName } =
    extractDatabaseResourceName(targetName);
  const instanceTitle = instance?.title || instanceName;
  const shouldShowInstance = showInstance && Boolean(instanceTitle);
  const shouldShowSeparator = shouldShowInstance && Boolean(databaseName);

  return (
    <div
      className={cn(
        "inline-flex max-w-full min-w-0 items-center",
        classes.root,
        className
      )}
    >
      {showEngine && instance && (
        <EngineIcon
          engine={instance.engine}
          className={cn("mr-1 shrink-0", classes.icon)}
        />
      )}
      {shouldShowInstance && (
        <EllipsisText
          text={instanceTitle}
          className={cn(
            "min-w-0 shrink-[2] truncate text-control-light",
            classes.instance
          )}
        />
      )}
      {shouldShowSeparator && (
        <span className="shrink-0 whitespace-pre text-control-light/80">
          {" / "}
        </span>
      )}
      {showEnvironment && environment?.title && (
        <EllipsisText
          text={environment.title}
          className={cn("mr-1 min-w-0 shrink", classes.environment)}
        >
          <EnvironmentLabel environment={environment} />
        </EllipsisText>
      )}
      <EllipsisText
        text={databaseName}
        className={cn("flex-1 truncate text-control", classes.database)}
      >
        <HighlightLabelText text={databaseName} keyword={keyword} />
      </EllipsisText>
    </div>
  );
}
