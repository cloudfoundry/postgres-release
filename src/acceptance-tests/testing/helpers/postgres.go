package helpers

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type PGData struct {
	Data PGCommon
	DBs  []PGConn
}

type PGCommon struct {
	Address     string
	Port        int
	SSLMode     string
	DefUser     string
	DefPassword string
}

type PGConn struct {
	TargetDB string
	user     string
	password string
	DB       *sql.DB
}

type PGSetting struct {
	Name    string `json:"name"`
	Setting string `json:"setting"`
	VarType string `json:"vartype"`
}
type PGDatabase struct {
	Name   string `json:"datname"`
	DBExts []PGDatabaseExtensions
	Tables []PGTable
}
type PGDatabaseExtensions struct {
	Name string `json:"extname"`
}
type PGTable struct {
	SchemaName     string `json:"schemaname"`
	TableName      string `json:"tablename"`
	TableOwner     string `json:"tableowner"`
	TableColumns   []PGTableColumn
	TableRowsCount PGCount
}
type PGTableColumn struct {
	ColumnName string `json:"column_name"`
	DataType   string `json:"data_type"`
	Position   int    `json:"ordinal_position"`
}
type PGCount struct {
	Num int `json:"count"`
}
type PGRole struct {
	Name        string `json:"rolname"`
	Super       bool   `json:"rolsuper"`
	Inherit     bool   `json:"rolinherit"`
	CreateRole  bool   `json:"rolcreaterole"`
	CreateDb    bool   `json:"rolcreatedb"`
	CanLogin    bool   `json:"rolcanlogin"`
	Replication bool   `json:"rolreplication"`
	ConnLimit   int    `json:"rolconnlimit"`
	ValidUntil  string `json:"rolvaliduntil"`
}

type PGOutputData struct {
	Roles     []PGRole
	Databases []PGDatabase
	Settings  map[string]string
}

const GetSettingsQuery = "SELECT * FROM pg_settings"
const ListRolesQuery = "SELECT * from pg_roles"
const ListDatabasesQuery = "SELECT datname from pg_database where datistemplate=false"
const ListDBExtensionsQuery = "SELECT extname from pg_extension"
const ConvertToDateCommand = "SELECT '%s'::timestamptz"
const ListTablesQuery = "SELECT * from pg_catalog.pg_tables where schemaname not like 'pg_%' and schemaname != 'information_schema'"
const ListTableColumnsQuery = "SELECT column_name, data_type, ordinal_position FROM information_schema.columns WHERE table_schema = '%s' AND table_name = '%s' order by ordinal_position asc"
const CountTableRowsQuery = "SELECT COUNT(*) FROM %s"
const QueryResultAsJson = "SELECT row_to_json(t) from (%s) as t;"

const NoConnectionAvailableErr = "No connections available"
const MissingDBAddressErr = "Database address not specified"
const MissingDBPortErr = "Database port not specified"
const MissingDefaultUserErr = "Default user not specified"
const MissingDefaultPasswordErr = "Default password not specified"

func GetFormattedQuery(query string) string {
	return fmt.Sprintf(QueryResultAsJson, query)
}

func NewPostgres(props PGCommon) (PGData, error) {
	var pg PGData
	if props.SSLMode == "" {
		props.SSLMode = "disable"
	}
	if props.Address == "" {
		return PGData{}, errors.New(MissingDBAddressErr)
	}
	if props.Port == 0 {
		return PGData{}, errors.New(MissingDBPortErr)
	}
	if props.DefUser == "" {
		return PGData{}, errors.New(MissingDefaultUserErr)
	}
	if props.DefPassword == "" {
		return PGData{}, errors.New(MissingDefaultPasswordErr)
	}
	pg.Data = props
	newConn, err := pg.OpenConnection("postgres", props.DefUser, props.DefPassword)
	if err != nil {
		return PGData{}, err
	}
	pg.DBs = append(pg.DBs, newConn)
	return pg, nil
}
func (pg PGData) OpenConnection(dbname string, user string, password string) (PGConn, error) {
	var newConn PGConn
	var err error
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", user, password, pg.Data.Address, pg.Data.Port, dbname, pg.Data.SSLMode)
	newConn.DB, err = sql.Open("postgres", url)
	if err != nil {
		return PGConn{}, err
	}
	err = newConn.DB.Ping()
	if err != nil {
		return PGConn{}, err
	}
	newConn.user = user
	newConn.password = password
	newConn.TargetDB = dbname

	return newConn, nil
}
func (pg PGData) GetDefaultConnection() PGConn {
	return pg.DBs[0]
}
func (pg PGData) GetDBConnection(dbname string) (PGConn, error) {
	return pg.GetDBConnectionForUser(dbname, "")
}
func (pg PGData) GetDBConnectionForUser(dbname string, user string) (PGConn, error) {
	if len(pg.DBs) == 0 {
		return PGConn{}, errors.New(NoConnectionAvailableErr)
	}
	var result PGConn
	for _, conn := range pg.DBs {
		if conn.TargetDB == dbname {
			if user == "" || conn.user == user {
				result = conn
				break
			}
		}
	}
	if (PGConn{}) == result {
		return PGConn{}, errors.New(NoConnectionAvailableErr)
	}
	return result, nil
}

func (pg PGConn) Run(query string) ([]string, error) {
	var result []string
	if rows, err := pg.DB.Query(GetFormattedQuery(query)); err != nil {
		return nil, err
	} else {
		defer rows.Close()
		for rows.Next() {
			var jsonRow string
			if err := rows.Scan(&jsonRow); err != nil {
				break
			}
			result = append(result, jsonRow)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (pg PGConn) Exec(query string) error {
	if _, err := pg.DB.Exec(query); err != nil {
		return err
	}
	return nil
}

func (pg PGData) CreateAndPopulateTables(dbName string, loadType LoadType) error {

	conn, err := pg.GetDBConnection(dbName)
	if err != nil {
		conn, err = pg.OpenConnection(dbName, pg.Data.DefUser, pg.Data.DefPassword)
		if err != nil {
			return err
		}
	}
	tables := GetSampleLoad(loadType)

	for _, table := range tables {
		err = conn.Exec(table.PrepareCreate())
		if err != nil {
			return err
		}
		txn, err := conn.DB.Begin()
		if err != nil {
			return err
		}

		stmt, err := txn.Prepare(table.PrepareStatement())
		if err != nil {
			return err
		}

		for i := 0; i < table.NumRows; i++ {
			_, err = stmt.Exec(table.PrepareRow(i)...)
			if err != nil {
				return err
			}
		}

		_, err = stmt.Exec()
		if err != nil {
			return err
		}

		err = stmt.Close()
		if err != nil {
			return err
		}

		err = txn.Commit()
		if err != nil {
			return err
		}
	}

	return err
}

func (pg PGData) ReadAllSettings() (map[string]string, error) {
	result := make(map[string]string)
	rows, err := pg.GetDefaultConnection().Run(GetSettingsQuery)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out := PGSetting{}
		err = json.Unmarshal([]byte(row), &out)
		if err != nil {
			return nil, err
		}
		result[out.Name] = out.Setting
	}
	return result, nil
}
func (pg PGData) ListDatabases() ([]PGDatabase, error) {
	var result []PGDatabase
	rows, err := pg.GetDefaultConnection().Run(ListDatabasesQuery)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out := PGDatabase{}
		err = json.Unmarshal([]byte(row), &out)
		if err != nil {
			return nil, err
		}
		result = append(result, out)
	}
	for idx, database := range result {
		result[idx].DBExts, err = pg.ListDatabaseExtensions(database.Name)
		if err != nil {
			return nil, err
		}
		result[idx].Tables, err = pg.ListDatabaseTables(database.Name)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (pg PGData) ListDatabaseExtensions(dbName string) ([]PGDatabaseExtensions, error) {
	conn, err := pg.GetDBConnection(dbName)
	if err != nil {
		conn, err = pg.OpenConnection(dbName, pg.Data.DefUser, pg.Data.DefPassword)
	}
	rows, err := conn.Run(ListDBExtensionsQuery)
	if err != nil {
		return nil, err
	}
	extensionsList := []PGDatabaseExtensions{}
	for _, row := range rows {
		out := PGDatabaseExtensions{}
		err = json.Unmarshal([]byte(row), &out)
		if err != nil {
			return nil, err
		}
		extensionsList = append(extensionsList, out)
	}
	return extensionsList, nil
}
func (pg PGData) ListDatabaseTables(dbName string) ([]PGTable, error) {
	conn, err := pg.GetDBConnection(dbName)
	if err != nil {
		conn, err = pg.OpenConnection(dbName, pg.Data.DefUser, pg.Data.DefPassword)
	}
	rows, err := conn.Run(ListTablesQuery)
	if err != nil {
		return nil, err
	}
	tableList := []PGTable{}
	for _, row := range rows {
		tableData := PGTable{}
		err = json.Unmarshal([]byte(row), &tableData)
		if err != nil {
			return nil, err
		}
		tableData.TableColumns = []PGTableColumn{}
		colRows, err := conn.Run(fmt.Sprintf(ListTableColumnsQuery, tableData.SchemaName, tableData.TableName))
		if err != nil {
			return nil, err
		}
		for _, colRow := range colRows {
			colData := PGTableColumn{}
			err = json.Unmarshal([]byte(colRow), &colData)
			if err != nil {
				return nil, err
			}
			tableData.TableColumns = append(tableData.TableColumns, colData)
		}
		countRows, err := conn.Run(fmt.Sprintf(CountTableRowsQuery, tableData.TableName))
		if err != nil {
			return nil, err
		}
		count := PGCount{}
		err = json.Unmarshal([]byte(countRows[0]), &count)
		if err != nil {
			return nil, err
		}
		tableData.TableRowsCount = count

		tableList = append(tableList, tableData)
	}
	return tableList, nil
}
func (pg PGData) ListRoles() ([]PGRole, error) {
	var result []PGRole
	rows, err := pg.GetDefaultConnection().Run(ListRolesQuery)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out := PGRole{}
		err = json.Unmarshal([]byte(row), &out)
		if err != nil {
			return nil, err
		}
		result = append(result, out)
	}
	return result, nil
}

func (pg PGData) ConvertToPostgresDate(inputDate string) (string, error) {
	type ConvertedDate struct {
		Date string `json:"timestamptz"`
	}
	result := ConvertedDate{}
	inputDate = strings.TrimLeft(inputDate, "'\"")
	inputDate = strings.TrimRight(inputDate, "'\"")
	rows, err := pg.GetDefaultConnection().Run(fmt.Sprintf(ConvertToDateCommand, inputDate))
	if err != nil {
		return "", err
	}
	err = json.Unmarshal([]byte(rows[0]), &result)
	if err != nil {
		return "", err
	}
	return result.Date, nil
}

func (pg PGData) GetData() (PGOutputData, error) {
	var result PGOutputData
	var err error
	result.Settings, err = pg.ReadAllSettings()
	if err != nil {
		return PGOutputData{}, err
	}
	result.Databases, err = pg.ListDatabases()
	if err != nil {
		return PGOutputData{}, err
	}
	result.Roles, err = pg.ListRoles()
	if err != nil {
		return PGOutputData{}, err
	}
	return result, nil
}

func (o PGOutputData) CopyData() (PGOutputData, error) {
	var to PGOutputData
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	dec := gob.NewDecoder(&buffer)
	err := enc.Encode(o)
	if err != nil {
		return PGOutputData{}, err
	}
	err = dec.Decode(&to)
	if err != nil {
		return PGOutputData{}, err
	}
	return to, nil
}
