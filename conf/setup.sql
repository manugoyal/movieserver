
CREATE DATABASE IF NOT EXISTS movieserver

USE movieserver

CREATE TABLE IF NOT EXISTS ips (
	address varchar(255) primary key,
	registered timestamp default current_timestamp,
	key registered(registered)
	)

CREATE TABLE IF NOT EXISTS movies(
	name varchar(255) primary key,
	downloads bigint unsigned default 0,
	key downloads(downloads)
	)

CREATE TABLE IF NOT EXISTS login(
	users varchar(255),
	passwords varchar(255),
	primary key (users, passwords)
	)
	

