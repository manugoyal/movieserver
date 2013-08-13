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
    confmoviefiles = [movie for movie in conf.movies
                      if not os.path.isdir(os.path.join(conf.moviepath, movie.name))]
    downloads = [random.randint(0, 10) for i in range(len(confmoviefiles))]
    for i in range(len(confmoviefiles)):
        movie = confmoviefiles[i]
        print movie
        for dnum in range(downloads[i]):
            req = requests.get(conf.serveraddress + conf.handlers.movie + movie.name)
            filetext = open(os.path.join(conf.moviepath, movie.name)).read()
            assert req.status_code == 200
            # It is encoded as a binary-stream, so req.content
            # contains the raw bytes
            assert filetext == req.content

    time.sleep(1)
    rows = conf.db.query("SELECT name, downloads FROM movies WHERE present=TRUE")
    for i in range(len(downloads)):
        assert [r for r in rows if r.name == confmoviefiles[i].name][0].downloads == downloads[i]

def test_bogus_file(conf):
    req = requests.get(conf.serveraddress + conf.handlers.movie + 'nonexistentfile.goober')
    assert req.status_code == 404

def test_directories(conf):
    """Gets a tar for each directory in conf.movies. Then it makes sure
    all the files are there in the tar and are equal"""
    confmoviedirs = [movie for movie in conf.movies
                     if os.path.isdir(os.path.join(conf.moviepath, movie.name))]
    for moviedir in confmoviedirs:
        req = requests.get(conf.serveraddress + conf.handlers.movie + moviedir.name)
        tfile = tarfile.open(mode='r', fileobj=StringIO.StringIO(req.content))
        if moviedir.name == '.':
            # It's the moviedirectory itself, so the tar archive will
            # have moviedir in it's name, so the tardir is
            # conf.moviepath minus the ending moviedir
            tardir = os.path.dirname(conf.moviepath)
        else:
            tardir = conf.moviepath
        for name in tfile.getnames():
            tarcontent = tfile.extractfile(name).read()
            syscontent = open(os.path.join(tardir, name)).read()
            assert tarcontent == syscontent
        # Asserts that the number of files in the tar equals the
        # number of files in the moviedir. This, along with the
        # assertion that every file in the tar is equal to a file in
        # the moviedir should prove that the tar is equal to the
        # moviedir
        moviedirfiles = [movie for movie in conf.movies
                         if not os.path.isdir(os.path.join(conf.moviepath, movie.name)) and
                         (movie.name.startswith(moviedir.name) or moviedir.name == '.')]
        assert len(moviedirfiles) == len(tfile.getnames())
