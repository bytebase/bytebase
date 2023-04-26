// Package mybatis defines the sql extractor for mybatis mapper xml.
package mybatis

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleMapper(t *testing.T) {
	testCases := []struct {
		name            string
		stmt            string
		wantRestoredSQL string
	}{
		{
			name: "simple mapper with one select",
			stmt: `
<mapper namespace="com.bytebase.test">
  <select id="selectUser" resultType="hashmap">
    SELECT * FROM user WHERE id = #{id}
  </select>
</mapper>`,
			wantRestoredSQL: "SELECT * FROM user WHERE id = #{id};\n",
		},
		{
			name: "simple mapper with select, update, insert, delete",
			stmt: `
<mapper namespace="com.bytebase.test">
  <select id="selectUser" resultType="hashmap">
    SELECT * FROM user WHERE id = #{id}
  </select>
  <update id="updateUser">
    UPDATE user SET name = #{name} WHERE id = #{id}
  </update>
  <insert id="insertUser">
    INSERT INTO user (name) VALUES (#{name})
  </insert>
  <delete id="deleteUser">
    DELETE FROM user WHERE id = #{id}
  </delete>
</mapper>`,
			wantRestoredSQL: `SELECT * FROM user WHERE id = #{id};
UPDATE user SET name = #{name} WHERE id = #{id};
INSERT INTO user (name) VALUES (#{name});
DELETE FROM user WHERE id = #{id};
`,
		},
	}

	for _, testCase := range testCases {
		node, err := Parse(testCase.stmt)
		assert.NoError(t, err)
		assert.NotNil(t, node)

		var stringsBuilder strings.Builder
		err = node.RestoreSQL(&stringsBuilder)
		assert.NoError(t, err)
		assert.Equal(t, testCase.wantRestoredSQL, stringsBuilder.String())
	}
}
