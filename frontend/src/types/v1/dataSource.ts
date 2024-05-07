import { v4 as uuidv4 } from "uuid";
import {
  DataSource,
  DataSourceType,
  DataSource_AuthenticationType,
} from "../proto/v1/instance_service";

export const DATASOURCE_ADMIN_USER_NAME = "bytebase";
export const DATASOURCE_READONLY_USER_NAME = `${DATASOURCE_ADMIN_USER_NAME}_readonly`;

export const emptyDataSource = () => {
  return DataSource.fromPartial({
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
