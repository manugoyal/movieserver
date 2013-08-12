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

package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"io/ioutil"
	"strings"
)

var (
	dbHandle      *sql.DB
	sqlStatements = make(map[string]string)
)

// Creates a *DB handle with user root to the given database
func connectRoot(dbName string) error {
	var err error
	dbHandle, err = sql.Open("mysql", "root@/"+dbName)
	if err != nil {
		return err
	}
	err = dbHandle.Ping()
	return err
}

const (
	stmtSep       = "----------"
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
	for _, stmt := range statements {
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
			glog.V(vLevel).Infof("Executing: %s", execstmt)
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
	sqlStatements["newMovie"] = "INSERT INTO movies(path, name) VALUES (?, ?) ON DUPLICATE KEY UPDATE present=TRUE"
	// addDownload increments the number of downloads for an
	// existing movie. If the movie isn't there, it won't throw an
	// error, but it will say that 0 rows were affected.
	sqlStatements["addDownload"] = "UPDATE movies SET downloads=downloads+1 WHERE path=? AND name=?"

	// getMovies selects all the movie names and downloads from
	// the movies table that are present. The three %s's are meant
	// for additional WHERE clauses, ORDER BY, and LIMIT
	sqlStatements["getMovies"] = "SELECT name, downloads FROM movies WHERE present = TRUE %s %s %s"

	// getMovieNum is the same as getMovies except it's a COUNT(*)
	// query. We don't need ORDER BY and LIMIT, though.
	sqlStatements["getMovieNum"] = "SELECT COUNT(*) FROM movies WHERE present = TRUE %s"

	// getUserAndPassword selects the row that matches a given
	// username-password combination
	sqlStatements["getUserAndPassword"] = "SELECT user from login WHERE user = ? AND password = ?"
}

// Sets up the schema, builds the query map, and sets all file entries
// which aren't in the *moviePath to present=False. This has to be
// done before the indexer starts indexing, so that the server doesn't
// accidentely return the wrong set of files to the client. Since it
// only has to be done once, we don't need to put it in the heartbeat
func startupDB() error {
	if err := setupSchema(); err != nil {
		return err
	}
	if _, err := dbHandle.Exec("UPDATE movies SET present=FALSE WHERE path != ?", *moviePath); err != nil {
		return err
	}
	buildSQLMap()
	return nil
}

// Closes the dbHandle
func cleanupDB() {
	glog.V(vLevel).Info("Cleaning up DB connection")
	if err := dbHandle.Close(); err != nil {
		glog.Errorf("Error during DB cleanup: %s", err)
	}
}
