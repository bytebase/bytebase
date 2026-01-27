import { create } from "@bufbuild/protobuf";
import { t } from "@/plugins/i18n";
import { extractGroupEmail } from "@/store/modules/v1/common";
import { type Group, GroupSchema } from "@/types/proto-es/v1/group_service_pb";
import { UNKNOWN_ID } from "../const";

export const unknownGroup = (name: string = ""): Group => {
  const group = create(GroupSchema, {
    name: `groups/${UNKNOWN_ID}`,
    title: t("common.unknown"),
  });
  if (name) {
    group.name = name;
    const email = extractGroupEmail(name);
    group.email = email;
    group.title = email.split("@")[0];
  }
  return group;
};
