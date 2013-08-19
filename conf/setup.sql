--#DROP DATABASE IF EXISTS movieserver
----------
CREATE DATABASE IF NOT EXISTS movieserver
----------
USE movieserver
----------
CREATE TABLE IF NOT EXISTS movies(
        path VARCHAR(767),
        name VARCHAR(767),
        downloads BIGINT UNSIGNED DEFAULT 0,
        PRIMARY KEY (path, name),
        KEY downloads(downloads)
        )
----------
CREATE TABLE IF NOT EXISTS login(
        user VARCHAR(255),
        password VARCHAR(255),
        PRIMARY KEY (user, password)
        )
