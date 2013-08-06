--#DROP DATABASE IF EXISTS movieserver
----------
CREATE DATABASE IF NOT EXISTS movieserver
----------
USE movieserver
----------
--#DROP TABLE IF EXISTS ips
----------
CREATE TABLE IF NOT EXISTS ips(
        address varchar(255) primary key,
        registered timestamp default current_timestamp,
        key registered(registered)
        )
----------
--#DROP TABLE IF EXISTS movies
----------
CREATE TABLE IF NOT EXISTS movies(
        name varchar(255) primary key,
        downloads bigint unsigned default 0,
        key downloads(downloads)
        )
