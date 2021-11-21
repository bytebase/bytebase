import { BookmarkID } from "./id";
import { Principal } from "./principal";

export type Bookmark = {
  id: BookmarkID;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  link: string;
};

export type BookmarkCreate = {
  // Domain specific fields
  name: string;
  link: string;
};
