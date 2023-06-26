import { DatabaseResource } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { Binding } from "@/types/proto/v1/project_service";

export type ComposedProjectMember = {
  user: User;
  bindingList: Binding[];
};

export interface SingleBinding {
  databaseResource?: DatabaseResource;
  expiration?: Date;
  description?: string;
  rawBinding: Binding;
}
