/*
Copyright 2013 Manu Goyal

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License.  You may obtain a copy of the
License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied.  See the License for the
specific language governing permissions and limitations under the License.
*/

// Utilitiy functions to setup and operate the mysql interface to the
// movieserver

package main;

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"strings"
	"github.com/golang/glog"
)

var (
	dbHandle *sql.DB
	sqlStatements = make(map[string]string)
)

// Creates a *DB handle with user root to the given database
func connectRoot(dbName string) error {
	var err error
	dbHandle, err = sql.Open("mysql", "root@/" + dbName)
	if err != nil {
		return err
	}
	err = dbHandle.Ping()
	return err
}

const (
	stmtSep = "----------"
	refreshPrefix = "--#"
)

// Runs the conf/setup.sql file. In the file, statements are separated
// by the stmtSep string. If the refreshSchema flag is true, we
// execute statements prefixed by refreshPrefix, otherwise we skip
// them. First it connects to no database to run the setup.sql
// statements, since they should create the movieserver database if
// that doesn't exist. Then it reconnects to the movieserver database
// (it can't rely on the USE database statement to use the database
// for subsequent statements run concurrently due to a bug in the
// mysql driver)
func setupSchema() error {
	if err := connectRoot(""); err != nil {
		return err
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
			glog.V(infoLevel).Infof("Executing: %s", execstmt)
			_, err := dbHandle.Exec(execstmt)
			if err != nil {
				return err
			}
		}
	}

	// Reconnects to the movieserver database
	if err := dbHandle.Close(); err != nil {
		return err
	}
	if err := connectRoot("movieserver"); err != nil {
		return err
	}
	return nil
}

// Adds some predefined SQL statements to a map
func buildSQLMap() {
	// newMovie adds a movie to the movies table. If the movie is
	// already there, it sets present to TRUE
	const newMovie = "INSERT INTO movies(path, name) VALUES (?, ?) ON DUPLICATE KEY UPDATE present=TRUE"
	sqlStatements["newMovie"] = newMovie;
	// addDownload increments the number of downloads for an
	// existing movie. If the movie isn't there, it won't throw an
	// error, but it will say that 0 rows were affected.
	const addDownload = "UPDATE movies SET downloads=downloads+1 WHERE path=? AND name=?"
	sqlStatements["addDownload"] = addDownload

	// getMovies selects all the movie names and downloads from
	// the movies table that are present.
	const getMovies = "SELECT name, downloads FROM movies WHERE present = TRUE"
	sqlStatements["getMovies"] = getMovies

	// getMovieNum is the same as getMovies except it's a COUNT(*) query
	const getMovieNum = "SELECT COUNT(*) FROM movies WHERE present = TRUE"
	sqlStatements["getMovieNum"] = getMovieNum

	// getAddr selects the ip addresses that matches a given value
	const getAddr = "SELECT address from ips WHERE address = ?"
	sqlStatements["getAddr"] = getAddr
}

// Initializes the dbHandle, sets up the schema, and compiles the SQL
func startupDB() error {
	if err := setupSchema(); err != nil {
		return err
	}
	buildSQLMap()
	return nil
}

// Closes the dbHandle
func cleanupDB() {
	glog.V(infoLevel).Info("Cleaning up DB connection")
	if err := dbHandle.Close(); err != nil {
		glog.Errorf("Error during DB cleanup: %s", err)
	}
}
