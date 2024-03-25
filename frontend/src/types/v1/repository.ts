import { Project } from "@/types/proto/v1/project_service";
import { ProjectGitOpsInfo } from "@/types/proto/v1/vcs_provider_service";

export interface ComposedRepository extends ProjectGitOpsInfo {
  project: Project;
}
