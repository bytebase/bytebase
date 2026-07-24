import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { isDarkTheme } from "@/modules/sql-editor/components/theme/derive";
import { useSQLEditorTheme } from "@/modules/sql-editor/components/theme/SQLEditorThemeScope";
import type { SQLEditorTreeNode } from "@/types";
import { DatabaseNode } from "./DatabaseNode";
import { InstanceNode } from "./InstanceNode";
import { LabelNode } from "./LabelNode";

type Props = {
  readonly node: SQLEditorTreeNode;
  readonly keyword: string;
  readonly checked: boolean;
  readonly checkDisabled?: boolean;
  readonly checkTooltip?: string;
  readonly onCheckedChange?: (checked: boolean) => void;
};

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/TreeNode/Label.vue.
 * Dispatches on `node.meta.type` to render the right leaf component; only
 * the `database` case consumes the checkbox/click callbacks.
 */
export function Label(props: Props) {
  const { node, keyword } = props;
  const type = node.meta.type;

  if (type === "instance") {
    return <InstanceNode node={node} keyword={keyword} />;
  }
  if (type === "environment") {
    return <EnvironmentNodeLabel node={node} />;
  }
  if (type === "database") {
    return (
      <DatabaseNode
        node={node}
        keyword={keyword}
        checked={props.checked}
        checkDisabled={props.checkDisabled}
        checkTooltip={props.checkTooltip}
        onCheckedChange={props.onCheckedChange}
      />
    );
  }
  if (type === "label") {
    return <LabelNode node={node} keyword={keyword} />;
  }
  return null;
}

function EnvironmentNodeLabel({ node }: Readonly<{ node: SQLEditorTreeNode }>) {
  const environment = (node as SQLEditorTreeNode<"environment">).meta.target;
  const theme = useSQLEditorTheme();
  const darkTheme = isDarkTheme(theme);

  return (
    <EnvironmentLabel
      environment={environment}
      link={false}
      styleOptions={
        darkTheme
          ? {
              defaultColorTextColor: theme.tokens["--color-accent-hover"],
              backgroundAlpha: 0.18,
            }
          : undefined
      }
    />
  );
}
