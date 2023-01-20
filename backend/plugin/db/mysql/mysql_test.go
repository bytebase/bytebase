package mysql

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransformDelimiter(t *testing.T) {
	tests := []struct {
		statement string
		want      string
	}{
		{
			statement: "CREATE TABLE t1(id INT PRIMAR KEY);",
			want:      "CREATE TABLE t1(id INT PRIMAR KEY);",
		},
		{
			statement: `
			DELIMITER ;;
			CREATE DEFINER=root@% TRIGGER upd_film AFTER UPDATE ON film FOR EACH ROW BEGIN
				IF (old.title != new.title) OR (old.description != new.description) OR (old.film_id != new.film_id)
				THEN
					UPDATE film_text
						SET title=new.title,
							description=new.description,
							film_id=new.film_id
					WHERE film_id=old.film_id;
				END IF;
			  END ;;
			DELIMITER ;
			`,
			want: `CREATE DEFINER=root@% TRIGGER upd_film AFTER UPDATE ON film FOR EACH ROW BEGIN
				IF (old.title != new.title) OR (old.description != new.description) OR (old.film_id != new.film_id)
				THEN
					UPDATE film_text
						SET title=new.title,
							description=new.description,
							film_id=new.film_id
					WHERE film_id=old.film_id;
				END IF;
			  END ;`,
		},
		{
			statement: `
			DELIMITER ;;
			CREATE DEFINER=root@% TRIGGER upd_film AFTER UPDATE ON film FOR EACH ROW BEGIN
				IF (old.title != new.title) OR (old.description != new.description) OR (old.film_id != new.film_id)
				THEN
					UPDATE film_text
						SET title=new.title,
							description=new.description,
							film_id=new.film_id
					WHERE film_id=old.film_id;
				END IF;
			  END ;;
			DELIMITER ;
			CREATE TABLE t1(id INT PRIMAR KEY);
			DELIMITER ;;
			CREATE DEFINER=root@% TRIGGER del_film AFTER DELETE ON film FOR EACH ROW BEGIN
				DELETE FROM film_text WHERE film_id = old.film_id;
			  END ;;
			DELIMITER ;
			`,
			want: `CREATE DEFINER=root@% TRIGGER upd_film AFTER UPDATE ON film FOR EACH ROW BEGIN
				IF (old.title != new.title) OR (old.description != new.description) OR (old.film_id != new.film_id)
				THEN
					UPDATE film_text
						SET title=new.title,
							description=new.description,
							film_id=new.film_id
					WHERE film_id=old.film_id;
				END IF;
			  END ;CREATE TABLE t1(id INT PRIMAR KEY);CREATE DEFINER=root@% TRIGGER del_film AFTER DELETE ON film FOR EACH ROW BEGIN
				DELETE FROM film_text WHERE film_id = old.film_id;
			  END ;`,
		},
	}
	a := require.New(t)
	for _, test := range tests {
		got, err := splitAndTransformDelimiter(test.statement)
		a.NoError(err)
		a.Len(got, 1)
		a.Equal(test.want, got[0])
	}
}

func TestTransformDelimiter_Truncate(t *testing.T) {
	var out bytes.Buffer
	for i := 0; i < 200000; i++ {
		_, err := out.WriteString("INSERT INTO hello VALUES (555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555555);")
		require.NoError(t, err)
	}
	statements := out.String()

	got, err := splitAndTransformDelimiter(statements)
	require.NoError(t, err)

	total := 0
	for _, trunk := range got {
		total += len(trunk)
	}
	require.Equal(t, 12, len(got))
	// Make sure all trunks add up.
	require.Equal(t, total, len(statements))
}
