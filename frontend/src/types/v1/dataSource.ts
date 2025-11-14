import { create } from "@bufbuild/protobuf";
import { v4 as uuidv4 } from "uuid";
// DataSource type used in function signatures, imported in value imports
import {
  DataSource_AuthenticationType,
  DataSourceSchema,
  DataSourceType,
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
