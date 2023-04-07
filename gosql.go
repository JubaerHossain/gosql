package gosql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

func GetColumns(params graphql.ResolveParams) string {
	fieldASTs := params.Info.FieldASTs
	var fields = make(map[string]interface{})
	for _, val := range fieldASTs {
		var cols []string
		for _, sel := range val.SelectionSet.Selections {
			field, ok := sel.(*ast.Field)
			if ok {
				if field.Name.Kind == "Name" {
					cols = append(cols, field.Name.Value)
				}
			}
		}
		fields[val.Name.Value] = cols
	}

	funclabel := fmt.Sprint(params.Info.Path.Key)
	cols := fields[funclabel].([]string) //
	selectColumn := strings.Join(cols, ",")
	return selectColumn

}

func ModelColumn(selectColumn string, v interface{}) ([]interface{}, error) {
	var columns []interface{}
	for _, column := range strings.Split(selectColumn, ",") {
		fieldName := strings.ToTitle(column[:1]) + column[1:]
		fieldValue := reflect.ValueOf(v).Elem().FieldByName(fieldName)
		if !fieldValue.IsValid() {
			return nil, fmt.Errorf("invalid field name: %s", fieldName)
		}
		columns = append(columns, fieldValue.Addr().Interface())
	}
	if len(columns) == 0 {
		return nil, errors.New("no columns selected")
	}
	return columns, nil
}

func BuildWhereClause(where map[string]interface{}) (string, []interface{}) {
	var whereClauses []string
	var whereArgs []interface{}

	for key, value := range where {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
		whereArgs = append(whereArgs, value)
	}
	return strings.Join(whereClauses, " AND "), whereArgs
}

func QueryModel(modelType reflect.Type, modelName string, params graphql.ResolveParams, db *sql.DB) (interface{}, error) {
	where, ok := params.Args["where"].(map[string]interface{})
	if !ok {
		where = make(map[string]interface{})
	}

	// Get the query parameters
	page, ok := params.Args["page"].(int)
	if !ok {
		page = 1
	}
	pageSize, ok := params.Args["pageSize"].(int)
	if !ok {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// Build the SQL query string
	selectColumn := GetColumns(params)
	whereClause, whereArgs := BuildWhereClause(where)
	if whereClause == "" {
		whereClause = "1 = 1"
	}
	sql := fmt.Sprintf("SELECT %s FROM %s WHERE %s ORDER BY id DESC LIMIT %d OFFSET %d;", selectColumn, modelName, whereClause, pageSize, offset)
	// Execute the query
	rows, err := db.Query(sql, whereArgs...)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	// Create a slice to hold the query results
	results := reflect.MakeSlice(reflect.SliceOf(modelType), 0, pageSize)

	// Loop through the query results and add them to the results slice
	for rows.Next() {
		// Create a new model instance
		model := reflect.New(modelType).Interface()

		// Get a list of pointers to the fields in the model struct
		columns, err := ModelColumn(selectColumn, model)
		if err != nil {
			return nil, err
		}

		// Scan the current row of data into the model struct fields
		err = rows.Scan(columns...)
		if err != nil {
			return nil, errors.New("no data found")
		}

		// Add the model to the results slice
		results = reflect.Append(results, reflect.ValueOf(model).Elem())
	}

	// Convert the results slice to an interface{} and return it
	return results.Interface(), nil
}

func FindByID(modelType reflect.Type, modelName string, params graphql.ResolveParams, db *sql.DB) (interface{}, error) {

	// Get the query parameters
	id, ok := params.Args["id"].(int)
	if !ok {
		return nil, errors.New("id is required")
	}
	// Build the SQL query string
	selectColumn := GetColumns(params)
	sql := fmt.Sprintf("SELECT %s FROM %s WHERE id = %d;", selectColumn, modelName, id)
	// Execute the query
	row := db.QueryRow(sql)
	// Create a new model instance
	model := reflect.New(modelType).Interface()

	// Get a list of pointers to the fields in the model struct
	columns, err := ModelColumn(selectColumn, model)
	if err != nil {
		return nil, err
	}
	// Scan the current row of data into the model struct fields
	err = row.Scan(columns...)
	if err != nil {
		return nil, errors.New("no data found")
	}
	return reflect.ValueOf(model).Elem().Interface(), nil
}

func QueryModelCount(modelName string, params graphql.ResolveParams, db *sql.DB) (interface{}, error) {

	// Build the SQL query string
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s;", modelName)

	// Execute the query
	row := db.QueryRow(sql)

	// Create a variable to hold the count
	var count int

	// Scan the current row of data into the count variable
	err := row.Scan(&count)
	if err != nil {
		return nil, errors.New("no data found")
	}

	// Convert the count to an interface{} and return it
	return count, nil
}

func CreateModel(modelType reflect.Type, modelName string, params graphql.ResolveParams, db *sql.DB) (interface{}, error) {

	// Get the model data from the GraphQL params
	model := params.Args["model"]

	// Convert the model data to a map
	modelMap, ok := model.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid model")
	}

	// Create a slice to hold the field names and a slice to hold the field values
	var fields []string
	var values []interface{}

	// Loop through the model fields and add them to the fields and values slices
	for key, value := range modelMap {
		fields = append(fields, key)
		values = append(values, value)
	}

	// Build the SQL query string
	fieldString := strings.Join(fields, ",")
	valueString := strings.Repeat("?,", len(fields))
	valueString = valueString[:len(valueString)-1]
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", modelName, fieldString, valueString)

	// Execute the query
	result, err := db.Exec(sql, values...)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	// Get the number of rows affected by the query
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, errors.New(err.Error())
	}

	if rowsAffected == 0 {
		return nil, errors.New("failed to create model")
	}

	// Return the newly created model data
	return model, nil
}

func UpdateModel(modelType reflect.Type, modelName string, params graphql.ResolveParams, db *sql.DB) (interface{}, error) {

	// Get the model data from the GraphQL params
	model := params.Args["model"]

	// Convert the model data to a map
	modelMap, ok := model.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid model")
	}

	// Create a slice to hold the field names and a slice to hold the field values
	var fields []string
	var values []interface{}

	// Loop through the model fields and add them to the fields and values slices
	for key, value := range modelMap {
		fields = append(fields, fmt.Sprintf("%s = ?", key))
		values = append(values, value)
	}

	// Build the SQL query string
	fieldString := strings.Join(fields, ",")
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?;", modelName, fieldString)

	// Execute the query
	result, err := db.Exec(sql, values...)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	// Get the ID of the updated model
	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.New(err.Error())
	}

	params.Args["id"] = id

	// Return the updated model
	return FindByID(modelType, modelName, params, db)
}

func DeleteModel(modelType reflect.Type, modelName string, params graphql.ResolveParams, db *sql.DB) (interface{}, error) {

	// Get the model ID from the GraphQL params
	id, ok := params.Args["id"].(int)
	if !ok {
		return nil, errors.New("id is required")
	}

	// Build the SQL query string
	sql := fmt.Sprintf("DELETE FROM %s WHERE id = ?;", modelName)

	// Execute the query
	result, err := db.Exec(sql, id)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	// Get the number of rows affected
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, errors.New(err.Error())
	}

	// Return the number of rows affected
	return rows, nil
}

func WhereModel(modelType reflect.Type, tableName string, params graphql.ResolveParams, where map[string]interface{}, db *sql.DB) (interface{}, error) {

	// Build the SQL query string
	selectColumn := GetColumns(params)
	whereClause, whereArgs := BuildWhereClause(where)
	sql := fmt.Sprintf("SELECT %s FROM %s WHERE %s;", selectColumn, tableName, whereClause)

	// Execute the query
	rows, err := db.Query(sql, whereArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Create a new slice to hold the model instances
	models := reflect.MakeSlice(reflect.SliceOf(modelType), 0, 0)

	// Loop through the query results and create a new model instance for each row
	for rows.Next() {
		// Create a new model instance
		model := reflect.New(modelType).Interface()

		// Get a list of pointers to the fields in the model struct
		columns, err := ModelColumn(selectColumn, model)
		if err != nil {
			return nil, err
		}

		// Scan the current row of data into the model struct fields
		err = rows.Scan(columns...)
		if err != nil {
			return nil, errors.New("no data found")
		}

		// Append the model instance to the models slice
		models = reflect.Append(models, reflect.ValueOf(model).Elem())
	}

	// Convert the models slice to an interface{} and return it
	return models.Interface(), nil
}
