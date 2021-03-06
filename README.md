Movie Server
============

A server to serve movies

Dependencies
=============

The server component is written entirely in Go, and requires version
&gt;= 1.1

The tests are written with Python 2.7 and the environment is set up with
virtualenv. To install virtualenv, run

    $ pip install virtualenv

Installation
=============

If you haven't set ``$GOPATH`` on your machine, you should run the
following commands before installing:

    $ mkdir -p $HOME/gocode
    $ export $GOPATH=$HOME/gocode
    $ export $PATH=$PATH:$GOPATH/bin

The ``$GOPATH`` location above is just an example. You can use any
directory. You may want to store the above commands in your
``.bashrc`` or another configuration file.

To install or update the movie server, run the following code:

    $ go get -u github.com/manugoyal/movieserver

This should install all of the server's dependencies as well.

To build the testing virtualenv, execute

    $ make testdeps

Usage
=====

To run the server:

    $ movieserver -path [location-name]=[path-to-directory] [-path ...]

There are a number of settings you can tweak via command line flags.
To get a complete description of the settings, run

    $ movieserver -help

To run the tests, execute

    $ make test

Note: On Macs, Python may not know where to find certain MySQL client
dylibs when importing the ``_mysql`` library. In order to fix this,
set the ``DYLD_LIBRARY_PATH`` environment variable to the location of
your MySQL installation's library directory (possibly
``/usr/local/mysql/lib``)


Licence
=========

Copyright 2013 Manu Goyal

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License.  You may obtain a copy of the
License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied.  See the License for the
specific language governing permissions and limitations under the License.
