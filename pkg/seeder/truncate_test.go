package seeder

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRunner_Truncate_QuotesForDialect(t *testing.T) {
	cases := []struct {
		name string
		opts []RunnerOption
		want string
	}{
		{"default is postgres", nil, `DELETE FROM "users"`},
		{"postgres", []RunnerOption{WithDialect(DialectPostgres)}, `DELETE FROM "users"`},
		{"mysql", []RunnerOption{WithDialect(DialectMySQL)}, "DELETE FROM `users`"},
		{"sqlite", []RunnerOption{WithDialect(DialectSQLite)}, `DELETE FROM "users"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			mock.ExpectExec("^" + regexp.QuoteMeta(c.want) + "$").WillReturnResult(sqlmock.NewResult(0, 3))

			r := NewRunner(NewRegistry(), db, nil, c.opts...)
			if err := r.Truncate("users"); err != nil {
				t.Fatalf("Truncate: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRunner_Truncate_WrapsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec("DELETE FROM").WillReturnError(errors.New("no such table"))

	err = NewRunner(NewRegistry(), db, nil, WithDialect(DialectMySQL)).Truncate("missing")
	if err == nil || !strings.Contains(err.Error(), `truncate table "missing"`) || !strings.Contains(err.Error(), "no such table") {
		t.Errorf("err = %v", err)
	}
}
