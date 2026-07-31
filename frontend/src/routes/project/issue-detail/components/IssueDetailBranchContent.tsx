import { useIssueDetailContext } from "../context/IssueDetailContext";
import { IssueDetailAccessGrantView } from "./IssueDetailAccessGrantView";
import { IssueDetailDatabaseChangeView } from "./IssueDetailDatabaseChangeView";
import { IssueDetailDatabaseCreateView } from "./IssueDetailDatabaseCreateView";
import { IssueDetailRoleGrantView } from "./IssueDetailRoleGrantView";

export function IssueDetailBranchContent({
  databaseChangeSelectedSpecId,
  onDatabaseChangeSelectedSpecIdChange,
}: {
  databaseChangeSelectedSpecId: string;
  onDatabaseChangeSelectedSpecIdChange: (specId: string) => void;
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
