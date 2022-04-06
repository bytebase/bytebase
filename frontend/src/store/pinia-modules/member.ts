import axios from "axios";
import { defineStore, storeToRefs } from "pinia";
import { watchEffect } from "vue";
import {
  Member,
  MemberId,
  MemberCreate,
  MemberPatch,
  MemberState,
  ResourceObject,
  PrincipalId,
  unknown,
  empty,
  EMPTY_ID,
  RoleType,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  member: ResourceObject,
  includedList: ResourceObject[]
): Member {
  const principal = getPrincipalFromIncludedList(
    member.relationships!.principal.data,
    includedList
  );
  principal.role = member.attributes.role as RoleType;

  return {
    ...(member.attributes as Omit<
      Member,
      "id" | "principal" | "creator" | "updater"
    >),
    id: parseInt(member.id),
    creator: getPrincipalFromIncludedList(
      member.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      member.relationships!.updater.data,
      includedList
    ),
    principal: principal,
  };
}

export const useMemberStore = defineStore("member", {
  state: (): MemberState => ({
    memberList: [],
  }),
  actions: {
    memberByPrincipalId(id: PrincipalId): Member {
      if (id == EMPTY_ID) {
        return empty("MEMBER") as Member;
      }

      return (
        this.memberList.find((item) => item.principal.id == id) ||
        (unknown("MEMBER") as Member)
      );
    },
    memberByEmail(email: string): Member {
      return (
        this.memberList.find((item) => item.principal.email == email) ||
        (unknown("MEMBER") as Member)
      );
    },
    setMemberList(memberList: Member[]) {
      this.memberList = memberList;
    },
    appendMember(newMember: Member) {
      this.memberList.push(newMember);
    },
    replaceMemberInList(updatedMember: Member) {
      const i = this.memberList.findIndex(
        (item: Member) => item.id == updatedMember.id
      );
      if (i !== -1) {
        this.memberList[i] = updatedMember;
      }
    },
    async fetchMemberList() {
      const data = (await axios.get(`/api/member`)).data;
      const memberList: Member[] = data.data.map((member: ResourceObject) => {
        return convert(member, data.included);
      });

      // sort the member list
      memberList.sort((a: Member, b: Member) => {
        if (a.createdTs === b.createdTs) {
          return a.id - b.id;
        }
        return a.createdTs - b.createdTs;
      });

      this.setMemberList(memberList);
      return memberList;
    },
    // Returns existing member if the principalId has already been created.
    async createdMember(newMember: MemberCreate) {
      const data = (
        await axios.post(`/api/member`, {
          data: {
            type: "MemberCreate",
            attributes: newMember,
          },
        })
      ).data;
      const createdMember = convert(data.data, data.included);

      this.appendMember(createdMember);

      return createdMember;
    },
    async patchMember({
      id,
      memberPatch,
    }: {
      id: MemberId;
      memberPatch: MemberPatch;
    }) {
      const data = (
        await axios.patch(`/api/member/${id}`, {
          data: {
            type: "memberPatch",
            attributes: memberPatch,
          },
        })
      ).data;
      const updatedMember = convert(data.data, data.included);

      this.replaceMemberInList(updatedMember);

      return updatedMember;
    },
    async deleteMemberById(id: MemberId) {
      await axios.delete(`/api/member/${id}`);

      const newList = this.memberList.filter((item: Member) => {
        return item.id != id;
      });

      this.setMemberList(newList);
    },
  },
});

export const useMemberList = () => {
  const store = useMemberStore();
  const { memberList } = storeToRefs(store);
  watchEffect(() => store.fetchMemberList());

  return memberList;
};
