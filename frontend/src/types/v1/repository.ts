import { Project } from "@/types/proto/v1/project_service";
import { ProjectGitOpsInfo } from "@/types/proto/v1/externalvs_service";

export interface ComposedRepository extends ProjectGitOpsInfo {
  project: Project;
}
