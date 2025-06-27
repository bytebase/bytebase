import { v4 as uuidv4 } from "uuid";
import { create } from "@bufbuild/protobuf";
// DataSource type used in function signatures, imported in value imports
import {
  DataSourceSchema,
  DataSourceType,
  DataSource_AuthenticationType,
} from "../proto-es/v1/instance_service_pb";

export const DATASOURCE_ADMIN_USER_NAME = "bytebase";
export const DATASOURCE_READONLY_USER_NAME = `${DATASOURCE_ADMIN_USER_NAME}_readonly`;

export const emptyDataSource = () => {
  return create(DataSourceSchema, {
    type: DataSourceType.ADMIN,
    id: uuidv4(),
    authenticationType: DataSource_AuthenticationType.PASSWORD,
    username: DATASOURCE_ADMIN_USER_NAME,
  });
};

export const unknownDataSource = () => {
  return {
    ...emptyDataSource(),
    title: "<<Unknown data source>>",
  };
};
