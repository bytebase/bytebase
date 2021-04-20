import axios from "axios";
import {
  Member,
  MemberId,
  MemberNew,
  MemberPatch,
  MemberState,
  ResourceObject,
  PrincipalId,
  unknown,
  empty,
  EMPTY_ID,
} from "../../types";

function convert(member: ResourceObject, rootGetters: any): Member {
  const creator = rootGetters["principal/principalById"](
    member.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    member.attributes.updaterId
  );
  return {
    ...(member.attributes as Omit<Member, "id" | "creator" | "updater">),
    id: member.id,
    creator,
    updater,
  };
}

const state: () => MemberState = () => ({
  memberList: [],
});

const getters = {
  memberList: (state: MemberState) => (): Member[] => {
    return state.memberList;
  },
  memberByPrincipalId: (state: MemberState) => (id: PrincipalId): Member => {
    if (id == EMPTY_ID) {
      return empty("MEMBER") as Member;
    }

    return (
      state.memberList.find((item) => item.principalId == id) ||
      (unknown("MEMBER") as Member)
    );
  },
};

const actions = {
  async fetchMemberList({ commit, rootGetters }: any) {
    const memberList = (await axios.get(`/api/member`)).data.data.map(
      (member: ResourceObject) => {
        return convert(member, rootGetters);
      }
    );

    commit("setMemberList", memberList);
    return memberList;
  },

  // Returns existing member if the principalId has already been created.
  async createdMember({ commit, rootGetters }: any, newMember: MemberNew) {
    const createdMember = convert(
      (
        await axios.post(`/api/member`, {
          data: {
            type: "membernew",
            attributes: newMember,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("appendMember", createdMember);

    return createdMember;
  },

  async patchMember(
    { commit, rootGetters }: any,
    { id, memberPatch }: { id: MemberId; memberPatch: MemberPatch }
  ) {
    const updatedMember = convert(
      (
        await axios.patch(`/api/member/${id}`, {
          data: {
            type: "memberpatch",
            attributes: memberPatch,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("replaceMemberInList", updatedMember);

    return updatedMember;
  },

  async deleteMemberById(
    { state, commit }: { state: MemberState; commit: any },
    id: MemberId
  ) {
    await axios.delete(`/api/member/${id}`);

    const newList = state.memberList.filter((item: Member) => {
      return item.id != id;
    });

    commit("setMemberList", newList);
  },
};

const mutations = {
  setMemberList(state: MemberState, memberList: Member[]) {
    state.memberList = memberList;
  },

  appendMember(state: MemberState, newMember: Member) {
    state.memberList.push(newMember);
  },

  replaceMemberInList(state: MemberState, updatedMember: Member) {
    const i = state.memberList.findIndex(
      (item: Member) => item.id == updatedMember.id
    );
    if (i != -1) {
      state.memberList[i] = updatedMember;
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
