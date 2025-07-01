import type { Release_File } from "@/types/proto-es/v1/release_service_pb";

export const getReleaseFileStatement = (file: Release_File) => {
  return new TextDecoder().decode(file.statement);
};
