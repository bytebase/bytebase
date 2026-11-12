import type { ReactNode } from "react";
import { Alert } from "@/components/ui/alert";

// A warning alert whose text passes body contrast: the shared warning variant
// sets title and body in the warning color itself.
export function NeonWarning({
  title,
  description,
  children,
}: {
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <Alert
      variant="warning"
      className="text-warning-hover"
      title={title}
      description={
        description === undefined ? undefined : (
          <span className="text-control">{description}</span>
        )
      }
    >
      {children}
    </Alert>
  );
}
