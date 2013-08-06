-- Copyright 2013 Manu Goyal
--
-- Licensed under the Apache License, Version 2.0 (the "License"); you may not use
-- this file except in compliance with the License.  You may obtain a copy of the
-- License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software distributed
-- under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
-- CONDITIONS OF ANY KIND, either express or implied.  See the License for the
-- specific language governing permissions and limitations under the License.

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
