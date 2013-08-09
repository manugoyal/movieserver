--#DROP DATABASE IF EXISTS movieserver
----------
CREATE DATABASE IF NOT EXISTS movieserver
----------
USE movieserver
----------
--#DROP TABLE IF EXISTS ips
----------
CREATE TABLE IF NOT EXISTS ips(
	address VARCHAR(255) PRIMARY KEY,
	registered TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	KEY registered(registered)
	)
----------
--#DROP TABLE IF EXISTS movies
----------
CREATE TABLE IF NOT EXISTS movies(
	path VARCHAR(767),
	name VARCHAR(767),
	downloads BIGINT UNSIGNED DEFAULT 0,
	present BOOL DEFAULT TRUE,
	PRIMARY KEY (path, name),
	KEY downloads(downloads),
	KEY present(present)
	)
----------
--#DROP TABLE IF EXISTS login
----------
CREATE TABLE IF NOT EXISTS login(
	user VARCHAR(255),
	password VARCHAR(255),
	PRIMARY KEY (user, password)
	)
