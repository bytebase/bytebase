import { Box, Info } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab-vue-state";
import {
  instanceV1SupportsExternalTable,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
} from "@/utils/v1/instance";
import type { AvailableAction } from "./actions";
import {
  ExternalTableIcon,
  FunctionIcon,
  PackageIcon,
  ProcedureIcon,
  SequenceIcon,
  TableLeafIcon,
  ViewIcon,
} from "./TreeNode/icons";

/**
 * React equivalent of Vue's
 * `useCurrentTabViewStateContext().availableActions`. Returns the same
 * ordered list, with the same per-instance support gating for
 * Sequences / External Tables / Packages, plus the trailing Diagram.
 *
 * Lifted to its own module (out of `SchemaPane.tsx`) so the AsidePanel
 * `ActionBar` and the SchemaPane right-click menu can share a single
 * source of truth.
 */
export function useAvailableActions(): AvailableAction[] {
  const { t } = useTranslation();
  const { instance: instanceRef } = useConnectionOfCurrentSQLEditorTab();
  const instance = useVueState(() => instanceRef.value);

  return useMemo(() => {
    const actions: AvailableAction[] = [
      {
        view: "INFO",
        title: t("common.info"),
        icon: <Info className="size-4" />,
      },
      { view: "TABLES", title: t("db.tables"), icon: <TableLeafIcon /> },
      { view: "VIEWS", title: t("db.views"), icon: <ViewIcon /> },
      { view: "FUNCTIONS", title: t("db.functions"), icon: <FunctionIcon /> },
      {
        view: "PROCEDURES",
        title: t("db.procedures"),
        icon: <ProcedureIcon />,
      },
    ];
    if (instanceV1SupportsSequence(instance)) {
      actions.push({
        view: "SEQUENCES",
        title: t("db.sequences"),
        icon: <SequenceIcon />,
      });
    }
    if (instanceV1SupportsExternalTable(instance)) {
      actions.push({
        view: "EXTERNAL_TABLES",
        title: t("db.external-tables"),
        icon: <ExternalTableIcon />,
      });
    }
    if (instanceV1SupportsPackage(instance)) {
      actions.push({
        view: "PACKAGES",
        title: t("db.packages"),
        icon: <PackageIcon />,
      });
    }
    actions.push({
      view: "DIAGRAM",
      title: t("schema-diagram.self"),
      icon: <Box className="size-4" />,
    });
    return actions;
  }, [t, instance]);
}
