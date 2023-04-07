# GoSQL

[![GoDoc](https://godoc.org/github.com/your-username/gosql?status.svg)](https://godoc.org/github.com/your-username/gosql)
[![Go Report Card](https://goreportcard.com/badge/github.com/your-username/gosql)](https://goreportcard.com/report/github.com/your-username/gosql)



GoSQL is a package that provides a simple way to query and manipulate SQL databases from GraphQL resolvers in Go.

Features - Query models by ID or with a where clause. - Specify which fields to select from the database using GraphQL queries. - Automatically convert database rows into Go struct instances. - Easily build a where clause with a map of column names and values. - Supports PostgreSQL, MySQL, and SQLite databases.

# Installation

To use this package in your Go project, you can install it using the go get command:

```golang
go get github.com/JubaerHossain/gosql
```

# Getting Started

To use GoSQL, first, you need to import the package:

```go
import "github.com/JubaerHossain/gosql"
```

Then, define your model struct and make sure that the field names match the column names in your database table:

```go

type User struct {
    ID        int
    FirstName string
    LastName  string
    Email     string
}


```
You can then use GoSQL to query the database and get the results as a slice of struct instances:

```go
func GetUser(params graphql.ResolveParams) (interface{}, error) {
    // Query the database for users
    users, err := gosql.QueryModel(reflect.TypeOf(User{}), "users", params)
    if err != nil {
        return nil, err
    }

    // Return the results
    return users, nil
}
```
You can also query a single record by ID:

```go
func GetUserByID(params graphql.ResolveParams) (interface{}, error) {
    // Query the database for a user by ID
    user, err := gosql.FindByID(reflect.TypeOf(User{}), "users", params)
    if err != nil {
        return nil, err
    }

    // Return the result
    return user, nil
}
```
Where Clauses
You can also build a where clause using a map of column names and values:

```go
func GetUsersWithLastName(params graphql.ResolveParams) (interface{}, error) {
    // Build the where clause
    where := make(map[string]interface{})
    where["last_name"] = params.Args["lastName"].(string)

    // Query the database for users
    users, err := gosql.QueryModel(reflect.TypeOf(User{}), "users", params, where)
    if err != nil {
        return nil, err
    }

    // Return the results
    return users, nil
}
```
Selecting Fields
You can specify which fields to select from the database using GraphQL queries:


```graphql
{
    users {
        firstName
        email
    }
}
```


```go
func GetUser(params graphql.ResolveParams) (interface{}, error) {
    // Get the columns to select from the query
    selectColumn := gosql.GetColumns(params)

    // Query the database for users
    users, err := gosql.QueryModel(reflect.TypeOf(User{}), "users", selectColumn, params)
    if err != nil {
        return nil, err
    }

    // Return the results
    return users, nil
}
```
Supported Databases
GoSQL supports PostgreSQL, MySQL, and SQLite databases. To connect to your database, you need to set the DB variable in the database package to a database handle.

```go
import "database/sql"
import _ "github.com/lib/pq"
import "lms/database"

func init() {
    db, err := sql.Open("postgres", "user=postgres dbname=lms sslmode=disable")
    if err != nil {
        panic(err)
    }

    database.DB = db
}
```
Contributing
Contributions are welcome! If you find a bug or want to add a feature, please open an issue or submit a pull request.

