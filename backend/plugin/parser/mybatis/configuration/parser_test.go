package configuration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractEnvironmentIDs(t *testing.T) {
	testCases := []struct {
		configuration string
		want          *Configuration
	}{
		{
			configuration: `
<?xml version="1.0" encoding="UTF-8" ?>
<!DOCTYPE configuration
	PUBLIC "-//mybatis.org//DTD Config 3.0//EN"
	"https://mybatis.org/dtd/mybatis-3-config.dtd">
<configuration>
	<environments default="prod">
	<environment id="prod">
		<transactionManager type="JDBC"/>
		<dataSource type="POOLED">
			<property name="driver" value="${driver}"/>
			<property name="url" value="jdbc:mysql://localhost:3306/test"/>
			<property name="username" value="${username}"/>
			<property name="password" value="${password}"/>
		</dataSource>
	</environment>
	<environment id="test">
		<transactionManager type="JDBC"/>
		<dataSource type="POOLED">
			<property name="driver" value="${driver}"/>
			<property name="url" value="jdbc:mysql://localhost:3306/test"/>
			<property name="username" value="${username}"/>
			<property name="password" value="${password}"/>
		</dataSource>
	</environment>
	</environments>
	<mappers>
	<mapper resource="org/mybatis/example/BlogMapper.xml"/>
	</mappers>
</configuration>
`,
			want: &Configuration{
				Environments: []Environment{
					{
						ID:             "prod",
						JDBCConnString: "jdbc:mysql://localhost:3306/test",
					},
					{
						ID:             "test",
						JDBCConnString: "jdbc:mysql://localhost:3306/test",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		got, err := ParseConfiguration(tc.configuration)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}
