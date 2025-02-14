import { SYSTEM_BOT_ID } from "../common";
import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { User, UserType } from "../proto/v1/user_service";
import { State } from "../proto/v1/common";

export const UNKNOWN_USER_NAME = `users/${UNKNOWN_ID}`;
export const SYSTEM_BOT_USER_NAME = `users/${SYSTEM_BOT_ID}`;

export const emptyUser = (): User => {
  return User.fromPartial({
    name: `users/${EMPTY_ID}`,
    state: State.ACTIVE,
    email: "",
    title: "",
    userType: UserType.USER,
  });
};

export const unknownUser = (): User => {
  return {
    ...emptyUser(),
    name: UNKNOWN_USER_NAME,
    title: "<<Unknown user>>",
  };
};

export const ALL_USERS_USER_ID = "2";
export const ALL_USERS_USER_EMAIL = "allUsers";
// Pseudo allUsers account.
export const allUsersUser = (): User => {
  return {
    ...emptyUser(),
    name: `users/${ALL_USERS_USER_ID}`,
    title: "All users",
    email: ALL_USERS_USER_EMAIL,
    userType: UserType.SYSTEM_BOT,
  };
};

export const filterUserListByKeyword = (userList: User[], keyword: string) => {
  keyword = keyword.trim().toLowerCase();
  if (!keyword) return userList;
  return userList.filter((user) => {
    return (
      user.title.toLowerCase().includes(keyword) ||
      user.email.toLowerCase().includes(keyword)
    );
  });
};

export const userBindingPrefix = "user:";

export const getUserEmailInBinding = (email: string) => {
  if (email === ALL_USERS_USER_EMAIL) {
    return email;
  }
  return `${userBindingPrefix}${email}`;
};

export const groupBindingPrefix = "group:";

export const getGroupEmailInBinding = (email: string) => {
  return `${groupBindingPrefix}${email}`;
};
