package helpers_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/cloudfoundry/postgres-release/src/acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var expectedcolumns = []string{"row_to_json"}
var genericError = fmt.Errorf("some error")

func convertQuery(query string) string {
	//return helpers.GetFormattedQuery(query)
	result := strings.Replace(helpers.GetFormattedQuery(query), ")", "\\)", -1)
	result = strings.Replace(result, "(", "\\(", -1)
	result = strings.Replace(result, ":", "\\:", -1)
	result = strings.Replace(result, "+", "\\+", -1)
	result = strings.Replace(result, "'", "\\'", -1)
	return strings.Replace(result, "*", "(.+)", -1)
}

func mockSettings(expected map[string]string, mocks map[string]sqlmock.Sqlmock) {
	if expected == nil {
		mocks["postgres"].ExpectQuery(convertQuery(helpers.GetSettingsQuery)).WillReturnError(genericError)
	} else {
		rows := sqlmock.NewRows(expectedcolumns)
		for key, value := range expected {
			ff := "{\"name\": \"%s\", \"setting\": \"%s\", \"some1\": \"%s\",\"vartype\": \"%s\"}"
			row := fmt.Sprintf(ff, key, value, "some0", "string")
			rows = rows.AddRow(row)
		}
		mocks["postgres"].ExpectQuery(convertQuery(helpers.GetSettingsQuery)).WillReturnRows(rows)
	}
}

func mockDatabases(expected []helpers.PGDatabase, mocks map[string]sqlmock.Sqlmock) {
	if expected == nil {
		mocks["postgres"].ExpectQuery(convertQuery(helpers.ListDatabasesQuery)).WillReturnError(genericError)
	} else {
		rows := sqlmock.NewRows(expectedcolumns)
		for _, elem := range expected {
			ff := "{\"datname\": \"%s\"}"
			row := fmt.Sprintf(ff, elem.Name)
			rows = rows.AddRow(row)
			extrows := sqlmock.NewRows(expectedcolumns)
			for _, elem := range elem.DBExts {
				xx := "{\"extname\": \"%s\"}"
				extrow := fmt.Sprintf(xx, elem.Name)
				extrows = extrows.AddRow(extrow)
			}
			mocks[elem.Name].ExpectQuery(convertQuery(helpers.ListDBExtensionsQuery)).WillReturnRows(extrows)
		}
		mocks["postgres"].ExpectQuery(convertQuery(helpers.ListDatabasesQuery)).WillReturnRows(rows)
	}
}
func mockRoles(expected []helpers.PGRole, mocks map[string]sqlmock.Sqlmock) error {
	if expected == nil {
		mocks["postgres"].ExpectQuery(convertQuery(helpers.ListRolesQuery)).WillReturnError(genericError)
	} else {
		rows := sqlmock.NewRows(expectedcolumns)
		for _, elem := range expected {
			row, err := json.Marshal(elem)
			if err != nil {
				return err
			}
			rows = rows.AddRow(row)
		}
		mocks["postgres"].ExpectQuery(convertQuery(helpers.ListRolesQuery)).WillReturnRows(rows)
	}
	return nil
}
func mockDate(current string, expected string, mocks map[string]sqlmock.Sqlmock) error {
	sqlCommand := convertQuery(fmt.Sprintf(helpers.ConvertToDateCommand, current))
	if expected == "" {
		mocks["postgres"].ExpectQuery(sqlCommand).WillReturnError(genericError)
	} else {
		row := fmt.Sprintf("{\"timestamptz\": \"%s\"}", expected)
		rows := sqlmock.NewRows(expectedcolumns).AddRow(row)
		mocks["postgres"].ExpectQuery(sqlCommand).WillReturnRows(rows)
	}
	return nil
}

var _ = Describe("Postgres", func() {
	Describe("Validate common data", func() {
		Context("Fail if common data is invalid", func() {
			It("Fail if no address provided", func() {
				props := helpers.PGCommon{
					Port:        10,
					DefUser:     "uu",
					DefPassword: "pp",
				}
				_, err := helpers.NewPostgres(props)
				Expect(err).To(MatchError(errors.New(helpers.MissingDBAddressErr)))
			})
			It("Fail if no port provided", func() {
				props := helpers.PGCommon{
					Address:     "bb",
					DefUser:     "uu",
					DefPassword: "pp",
				}
				_, err := helpers.NewPostgres(props)
				Expect(err).To(MatchError(errors.New(helpers.MissingDBPortErr)))
			})
			It("Fail if no default user provided", func() {
				props := helpers.PGCommon{
					Address:     "bb",
					Port:        10,
					DefPassword: "pp",
				}
				_, err := helpers.NewPostgres(props)
				Expect(err).To(MatchError(errors.New(helpers.MissingDefaultUserErr)))
			})
			It("Fail if no default password provided", func() {
				props := helpers.PGCommon{
					Address: "bb",
					Port:    10,
					DefUser: "uu",
				}
				_, err := helpers.NewPostgres(props)
				Expect(err).To(MatchError(errors.New(helpers.MissingDefaultPasswordErr)))
			})
			It("Fail if incorrect data provided", func() {
				props := helpers.PGCommon{
					Address:     "bb",
					Port:        10,
					DefUser:     "uu",
					DefPassword: "pp",
				}
				_, err := helpers.NewPostgres(props)
				Expect(err).To(MatchError(ContainSubstring("no such host")))
			})
		})
	})
	Describe("Run read-only queries", func() {
		var (
			mocks map[string]sqlmock.Sqlmock
			pg    *helpers.PGData
		)

		BeforeEach(func() {
			mocks = make(map[string]sqlmock.Sqlmock)
			db, mock, err := sqlmock.New()
			Expect(err).NotTo(HaveOccurred())
			mocks["postgres"] = mock
			db1, mock1, err := sqlmock.New()
			Expect(err).NotTo(HaveOccurred())
			mocks["db1"] = mock1
			db2, mock2, err := sqlmock.New()
			Expect(err).NotTo(HaveOccurred())
			mocks["db2"] = mock2
			pg = &helpers.PGData{
				Data: helpers.PGCommon{},
				DBs: []helpers.PGConn{
					helpers.PGConn{
						DB:       db,
						TargetDB: "postgres",
					},
					helpers.PGConn{
						DB:       db1,
						TargetDB: "db1",
					},
					helpers.PGConn{
						DB:       db2,
						TargetDB: "db2",
					},
				},
			}
		})
		AfterEach(func() {
			for _, conn := range pg.DBs {
				conn.DB.Close()
			}
		})
		Context("Run a generic query", func() {
			It("Returns all the lines", func() {
				expected := []string{
					"{\"name\": \"pgadmin1\", \"role\": \"admin\"}",
					"{\"name\": \"pgadmin2\", \"role\": \"admin\"}",
				}
				rows := sqlmock.NewRows(expectedcolumns).
					AddRow(expected[0]).
					AddRow(expected[1])
				query := "SELECT name,role FROM table"
				mocks["postgres"].ExpectQuery(convertQuery(query)).WillReturnRows(rows)
				result, err := pg.GetDefaultConnection().Run(query)
				Expect(err).NotTo(HaveOccurred())
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(result).To(Equal(expected))
			})
			It("Properly reports a failure", func() {
				query := "SELECT name,role FROM table"
				mocks["postgres"].ExpectQuery(convertQuery(query)).WillReturnError(genericError)
				_, err := pg.GetDefaultConnection().Run(query)
				Expect(err).To(MatchError(genericError))
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})
		Context("Fail to retrieve env info", func() {
			It("Fails to read pg_settings", func() {
				mockSettings(nil, mocks)
				_, err := pg.ReadAllSettings()
				Expect(err).To(MatchError(genericError))
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
			})
			It("Fails to list databases", func() {
				mockDatabases(nil, mocks)
				_, err := pg.ListDatabases()
				Expect(err).To(MatchError(genericError))
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
			})
			It("Fails to list roles", func() {
				err := mockRoles(nil, mocks)
				Expect(err).NotTo(HaveOccurred())
				_, err = pg.ListRoles()
				Expect(err).To(MatchError(genericError))
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
			})
			It("Fails to convert date to postgres date", func() {
				err := mockDate("xxx", "", mocks)
				Expect(err).NotTo(HaveOccurred())
				_, err = pg.ConvertToPostgresDate("xxx")
				Expect(err).To(MatchError(genericError))
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})
		Context("Correctly retrieve env info", func() {
			It("Correctly read pg_settings", func() {
				expected := map[string]string{
					"a1": "a2",
					"b1": "b2",
				}
				mockSettings(expected, mocks)
				result, err := pg.ReadAllSettings()
				Expect(err).NotTo(HaveOccurred())
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(result).NotTo(BeZero())
				Expect(result).To(Equal(expected))
			})
			It("Correctly lists databases without extensions", func() {
				expected := []helpers.PGDatabase{
					helpers.PGDatabase{
						Name:   "db1",
						DBExts: []helpers.PGDatabaseExtensions{},
					},
					helpers.PGDatabase{
						Name:   "db2",
						DBExts: []helpers.PGDatabaseExtensions{},
					},
				}
				mockDatabases(expected, mocks)
				result, err := pg.ListDatabases()
				Expect(err).NotTo(HaveOccurred())
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				if err = mocks["db1"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				if err = mocks["db2"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(result).To(Equal(expected))
			})
			It("Correctly lists databases with extensions", func() {
				expected := []helpers.PGDatabase{
					helpers.PGDatabase{
						Name: "db1",
						DBExts: []helpers.PGDatabaseExtensions{
							helpers.PGDatabaseExtensions{
								Name: "exta",
							},
							helpers.PGDatabaseExtensions{
								Name: "extb",
							},
						},
					},
				}
				mockDatabases(expected, mocks)
				result, err := pg.ListDatabases()
				Expect(err).NotTo(HaveOccurred())
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				if err = mocks["db1"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(result).To(Equal(expected))
			})
			It("Correctly lists roles with properties", func() {
				expected := []helpers.PGRole{
					helpers.PGRole{
						Name:        "role1",
						Super:       true,
						Inherit:     false,
						CreateRole:  false,
						CreateDb:    true,
						CanLogin:    true,
						Replication: false,
						ConnLimit:   10,
						ValidUntil:  "",
					},
					helpers.PGRole{
						Name:        "role2",
						Super:       false,
						Inherit:     true,
						CreateRole:  true,
						CreateDb:    false,
						CanLogin:    false,
						Replication: true,
						ConnLimit:   100,
						ValidUntil:  "xxx",
					},
				}
				err := mockRoles(expected, mocks)
				Expect(err).NotTo(HaveOccurred())
				result, err := pg.ListRoles()
				Expect(err).NotTo(HaveOccurred())
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(result).To(Equal(expected))
			})
			It("Correctly retrieve environment data", func() {
				expected := helpers.PGOutputData{
					Roles: []helpers.PGRole{
						helpers.PGRole{
							Name:      "pgadmin",
							CanLogin:  true,
							ConnLimit: 20,
						},
					},
					Databases: []helpers.PGDatabase{
						helpers.PGDatabase{
							Name:   "db1",
							DBExts: []helpers.PGDatabaseExtensions{},
						},
					},
					Settings: map[string]string{
						"max_connections": "30",
					},
				}
				mockSettings(expected.Settings, mocks)
				mockDatabases(expected.Databases, mocks)
				err := mockRoles(expected.Roles, mocks)
				Expect(err).NotTo(HaveOccurred())
				result, err := pg.GetData()
				Expect(err).NotTo(HaveOccurred())
				if err = mocks["postgres"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				if err = mocks["db1"].ExpectationsWereMet(); err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(result).NotTo(BeZero())
				Expect(result).To(Equal(expected))
			})
			It("Correctly convert date to postgres date", func() {
				input := "May 5 12:00:00 2017 +1"
				expected := "2017-05-05 11:00:00"
				err := mockDate(input, expected, mocks)
				Expect(err).NotTo(HaveOccurred())
				result, err := pg.ConvertToPostgresDate(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(expected))
			})
			It("Correctly convert date with quotes to postgres date", func() {
				input := "May 5 12:00:00 2017 +1"
				inputQuotes := fmt.Sprintf("'%s'", input)
				expected := "2017-05-05 11:00:00"
				err := mockDate(input, expected, mocks)
				Expect(err).NotTo(HaveOccurred())
				result, err := pg.ConvertToPostgresDate(inputQuotes)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(expected))
			})
		})
	})
})
