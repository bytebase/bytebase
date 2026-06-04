import { DatabaseGroupForm } from "@/react/components/DatabaseGroupForm";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { router } from "@/react/router";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";

export function ProjectDatabaseGroupCreatePage({
  projectId,
}: {
  projectId: string;
}) {
  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;
  const project = useProjectByName(projectName);

  if (!project) return null;

  return (
    <DatabaseGroupForm
      className="py-4 h-full"
      readonly={false}
      project={project}
      onDismiss={() => router.back()}
      onCreated={(databaseGroupName) => {
        router.push({
          name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
          params: { databaseGroupName },
        });
      }}
    />
  );
}
