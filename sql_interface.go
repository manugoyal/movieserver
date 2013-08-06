// Utilitiy functions to setup and operate the mysql interface to the
// movieserver

package main;

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"strings"
	"log"
)

var (
	dbHandle *sql.DB
	insertStatements = make(map[string]*sql.Stmt)
	selectStatements = make(map[string]*sql.Stmt)
)

// Creates a *DB handle with user "root" and no other connection
// params. Also compiles the SQL statements
func connectRoot() error {
	var err error
	dbHandle, err = sql.Open("mysql", "root@/movieserver")
	if err != nil {
		return err
	}
	err = dbHandle.Ping()
	return err
}

// Compiles the predefined SQL statements
func compileSQL() error {
	var err error
	// newMovie adds a movie to the movies table. If the movie is
	// already there, it does nothing
	const newMovie = "INSERT INTO movies(name) VALUES (?) ON DUPLICATE KEY UPDATE name=name"
	if insertStatements["newMovie"], err = dbHandle.Prepare(newMovie); err != nil {
		return err
	}
	// addDownload increments the number of downloads for an
	// existing movie. If the movie isn't there, it won't throw an
	// error, but it will say that 0 rows were affected.
	const addDownload = "UPDATE movies SET downloads=downloads+1 WHERE name=(?)"
	if insertStatements["addDownload"], err = dbHandle.Prepare(addDownload); err != nil {
		return err
	}

	// getNames selects all the movie names from the movies table
	const getNames = "SELECT name FROM movies"
	if selectStatements["getNames"], err = dbHandle.Prepare(getNames); err != nil {
		return err
	}

	return nil
}

const (
	stmtSep = "----------"
	refreshPrefix = "--#"
)

// Runs the conf/setup.sql file. In the file, statements are separated
// by the stmtSep string. If the refreshSchema flag is true, we
// execute statements prefixed by refreshPrefix, otherwise we skip
// them
func setupSchema() error {
	if dbHandle == nil || dbHandle.Ping() != nil {
		panic("dbHandle isn't set up yet")
	}
	setupBytes, err := ioutil.ReadFile(*srcPath + "/conf/setup.sql")
	if err != nil {
		return err
	}
	statements := strings.Split(string(setupBytes), stmtSep)
	for _, stmt := range(statements) {
		stmt = strings.TrimSpace(stmt)
		execstmt := ""
		if strings.HasPrefix(stmt, refreshPrefix) {
			if *refreshSchema {
				execstmt = stmt[len(refreshPrefix):]
			}
		} else {
			execstmt = stmt
		}
		if len(execstmt) > 0 {
			log.Printf("Executing: %s", execstmt)
			_, err := dbHandle.Exec(execstmt)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Closes the sql statements and the dbHandle
func cleanupDB() {
	log.Print("Cleaning up DB connection")
	const DBErrmsg = "Error during DB cleanup: %s"
	var err error
	for _, stmt := range(insertStatements) {
		if err = stmt.Close(); err != nil {
			log.Printf(DBErrmsg, err)
		}
	}
	for _, stmt := range(selectStatements) {
		if err = stmt.Close(); err != nil {
			log.Printf(DBErrmsg, err)
		}
	}

	if err = dbHandle.Close(); err != nil {
		log.Printf(DBErrmsg, err)
	}
}
