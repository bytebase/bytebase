import { User } from "@/types/proto/v1/auth_service";

export type ComposedProjectMember = {
  user: User;
  roleList: string[];
};
