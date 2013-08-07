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

CREATE TABLE IF NOT EXISTS setup(
	users varchar(255) primary key NOT NULL,
	passwords varchar(255) NOT NULL,
	key users(users)
	)
	
