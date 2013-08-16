# Tests the movie handler
import requests
import random
import pytest
import time
import os.path
import StringIO
import tarfile

def setup_module():
    random.seed()

@pytest.fixture(autouse=True)
def cleardownloads(request, conf):
    conf.db.execute("UPDATE movies SET downloads=0 WHERE present=TRUE")

def test_files(conf):
    for tableKey, path in conf.paths.iteritems():
        confmoviefiles = [movie for movie in conf.movies[tableKey]
                          if not os.path.isdir(os.path.join(path, movie.name))]
        downloads = [random.randint(0, 10) for i in range(len(confmoviefiles))]
        for i in range(len(confmoviefiles)):
            movie = confmoviefiles[i]
            for dnum in range(downloads[i]):
                req = requests.get(conf.serveraddress + conf.handlers.movie[tableKey] + movie.name)
                filetext = open(os.path.join(path, movie.name)).read()
                assert req.status_code == 200
                # It is encoded as a binary-stream, so req.content
                # contains the raw bytes
                assert filetext == req.content

        time.sleep(1)
        for i in range(len(downloads)):
            rows = conf.db.query("SELECT name, downloads FROM movies WHERE present=TRUE AND path=%s AND name=%s", path, confmoviefiles[i].name)
            assert len(rows) == 1
            assert rows[0].downloads == downloads[i]

def test_bogus_file(conf):
    for tableKey in conf.paths.iterkeys():
        req = requests.get(conf.serveraddress + conf.handlers.movie[tableKey] + 'nonexistentfile.goober')
        assert req.status_code == 404

def test_directories(conf):
    """Gets a tar for each directory in conf.movies. Then it makes sure
    all the files are there in the tar and are equal"""
    for tableKey, path in conf.paths.iteritems():
        confmoviedirs = [movie for movie in conf.movies[tableKey]
                         if os.path.isdir(os.path.join(path, movie.name))]
        for moviedir in confmoviedirs:
            req = requests.get(conf.serveraddress + conf.handlers.movie[tableKey] + moviedir.name)
            assert req.status_code == 200
            tfile = tarfile.open(mode='r', fileobj=StringIO.StringIO(req.content))
            for tname in tfile.getnames():
                tarcontent = tfile.extractfile(tname).read()
                tardir = os.path.dirname(os.path.normpath(os.path.join(path, moviedir.name)))
                syscontent = open(os.path.join(tardir, tname)).read()
                assert tarcontent == syscontent
            # Asserts that the number of files in the tar equals the
            # number of files in the moviedir. This, along with the
            # assertion that every file in the tar is equal to a file in
            # the moviedir should prove that the tar is equal to the
            # moviedir
            moviedirfiles = [movie for movie in conf.movies[tableKey]
                             if not os.path.isdir(os.path.join(conf.paths[tableKey], movie.name)) and
                             (movie.name.startswith(moviedir.name) or moviedir.name == '.')]
            assert len(moviedirfiles) == len(tfile.getnames())
