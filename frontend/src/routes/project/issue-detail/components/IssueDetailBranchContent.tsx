import { useIssueDetailContext } from "../context/IssueDetailContext";
import { IssueDetailAccessGrantView } from "./IssueDetailAccessGrantView";
import { IssueDetailDatabaseChangeView } from "./IssueDetailDatabaseChangeView";
import { IssueDetailDatabaseCreateView } from "./IssueDetailDatabaseCreateView";
import { IssueDetailDatabaseExportView } from "./IssueDetailDatabaseExportView";
import { IssueDetailRoleGrantView } from "./IssueDetailRoleGrantView";

export function IssueDetailBranchContent({
  databaseChangeSelectedSpecId,
  databaseExportExecutionHistoryExpanded,
  databaseExportTasksExpanded,
  onDatabaseChangeSelectedSpecIdChange,
  onDatabaseExportExecutionHistoryExpandedChange,
  onDatabaseExportTasksExpandedChange,
}: {
  databaseChangeSelectedSpecId: string;
  databaseExportExecutionHistoryExpanded: boolean;
  databaseExportTasksExpanded: boolean;
  onDatabaseChangeSelectedSpecIdChange: (specId: string) => void;
  onDatabaseExportExecutionHistoryExpandedChange: (expanded: boolean) => void;
  onDatabaseExportTasksExpandedChange: (expanded: boolean) => void;
}) {
  const page = useIssueDetailContext();

  if (page.issueType === "ROLE_GRANT") {
    return <IssueDetailRoleGrantView />;
  }

  if (page.issueType === "ACCESS_GRANT") {
    return <IssueDetailAccessGrantView />;
  }

  if (page.issueType === "CREATE_DATABASE") {
    return <IssueDetailDatabaseCreateView />;
  }

  if (page.issueType === "EXPORT_DATA") {
    return (
      <IssueDetailDatabaseExportView
        executionHistoryExpanded={databaseExportExecutionHistoryExpanded}
        onExecutionHistoryExpandedChange={
          onDatabaseExportExecutionHistoryExpandedChange
        }
        onTasksExpandedChange={onDatabaseExportTasksExpandedChange}
        tasksExpanded={databaseExportTasksExpanded}
      />
    );
  }

  if (page.issueType === "DATABASE_CHANGE") {
    return (
      <IssueDetailDatabaseChangeView
        onSelectedSpecIdChange={onDatabaseChangeSelectedSpecIdChange}
        selectedSpecId={databaseChangeSelectedSpecId}
      />
    );
  }

  return null;
}
