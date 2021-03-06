# Tests the movietable handler

import requests
import fnmatch
import random

def setup_module():
    random.seed()

# Makes sure the server indexes all the files in the moviedir
def test_handler_files(conf):
    for tableKey, path in conf.paths.iteritems():
        rows = conf.db.query("SELECT path, name FROM movies WHERE path = %s", path)
        badpaths = [r.path for r in rows if r.path != path]
        assert len(badpaths) == 0
        assert set([r.name for r in rows]) == set([movie.name for movie in conf.movies[tableKey]])

# Given the parameters (which aren't in string form yet), it runs a
# table query and figures out the right asserts to make. Given a page
# argument when setPageOutOfBounds is False, it will fetch all the
# pages up to and including that page and test those, not just the
# specified page. It runs the query on all of the movie directories.
def param_query(params, conf, setPageOutOfBounds=False):
    for tableKey in conf.paths.iterkeys():
        if 'q' in params:
            # The server runs a prefix match, so we add a star
            confmovies = [movie for movie in conf.movies[tableKey]
                          if fnmatch.fnmatch(movie.name, params['q'] + '*')]
        else:
            confmovies = [movie for movie in conf.movies[tableKey]]

        # Sets a page that is guaranteed to be out of bounds
        if setPageOutOfBounds:
            params['page'] = len(confmovies) + 1

        confmovienames = set([movie.name for movie in confmovies])

        if 'per_page' in params:
            resp_len = min(params['per_page'], len(confmovies))
        else:
            resp_len = len(confmovies)

        strparams = {k: str(v) for k, v in params.iteritems()}

        results = []

        if setPageOutOfBounds:
            assert 'page' in params and 'per_page' in params
            req = requests.get(conf.serveraddress + conf.handlers.table[tableKey], params=strparams)
            resp = req.json()
            results = resp[1]

            assert resp[0]['total_entries'] == len(confmovies)
            assert resp[0]['page'] == 1
            assert resp[0]['per_page'] == params['per_page']
            assert len(resp[1]) == resp_len

            nameset = set([movie['name'] for movie in resp[1]])
            assert nameset.issubset(confmovienames)
        else:
            if 'page' in params:
                assert 'per_page' in params
                allnames = []
                for i in range(params['page']):
                    strparams['page'] = str(i + 1)
                    req = requests.get(conf.serveraddress + conf.handlers.table[tableKey], params=strparams)
                    resp = req.json()
                    results.extend(resp[1])

                    assert resp[0]['total_entries'] == len(confmovies)
                    assert len(resp[1]) == resp_len

                    nameset = set([movie['name'] for movie in resp[1]])
                    allnames.extend(nameset)
                    assert nameset.issubset(confmovienames)
                assert len(allnames) == len(set(allnames))
                assert set(allnames).issubset(confmovienames)
            else:
                assert 'per_page' not in params
                req = requests.get(conf.serveraddress + conf.handlers.table[tableKey], params=strparams)
                resp = req.json()
                results = resp[1]

                assert resp[0]['total_entries'] == len(confmovies)
                assert len(resp[1]) == resp_len

                nameset = set([movie['name'] for movie in resp[1]])
                assert nameset == confmovienames

        if 'sort_by' in params:
            assert 'order' in params
            sortcol = params['sort_by']
            sortedmoviecols = sorted([movie[sortcol] for movie in confmovies])
            if params['order'] == 'desc':
                sortedmoviecols.reverse()
            assert sortedmoviecols[:len(results)] == [movie[sortcol] for movie in results]

def test_noargs_json(conf):
    param_query({}, conf)

def test_pone_ppone_json(conf):
    param_query({'page': 1, 'per_page': 1}, conf)

def test_pone_pptwo_json(conf):
    param_query({'page': 1, 'per_page': 2}, conf)

def test_ptwo_ppone_json(conf):
    param_query({'page': 2, 'per_page': 1}, conf)

def test_qdogs(conf):
    param_query({'q': '.dogs'}, conf)

def test_qstarx(conf):
    param_query({'q': '*x'}, conf)

def test_qslashx(conf):
    param_query({'q': '*/x'}, conf)

def test_qdogs_pone_ppone(conf):
    param_query({'q': '.dogs', 'page': 1, 'per_page': 1}, conf)

def test_qstarx_ptwo_ppone(conf):
    param_query({'q': '*x', 'page': 2, 'per_page': 1}, conf)

def test_qstarx_ptwo_pptwo(conf):
    param_query({'q': '*x', 'per_page': 2}, conf, setPageOutOfBounds=True)

def test_qslashx_ptwo_pptwo(conf):
    param_query({'q': '*/x', 'per_page': 2}, conf, setPageOutOfBounds=True)

def test_orderby_name(conf):
    param_query({'sort_by': 'name', 'order': 'asc', 'page': 1, 'per_page': 3}, conf)
    param_query({'sort_by': 'name', 'order': 'desc', 'page': 1, 'per_page': 3}, conf)

def test_orderby_downloads(conf):
    downloads = {}
    # Sets the download num for each file both in the database and
    # in conf.movies
    for tableKey, path in conf.paths.iteritems():
        downloads[tableKey] = [random.randint(0, 10000) for i in range(len(conf.movies[tableKey]))]
        for i in range(len(conf.movies[tableKey])):
            conf.db.execute('UPDATE movies SET downloads=%s WHERE path=%s AND name=%s',
                            downloads[tableKey][i], path, conf.movies[tableKey][i].name)
            conf.movies[tableKey][i]['downloads'] = downloads[tableKey][i]

    param_query({'sort_by': 'downloads', 'order': 'asc', 'page': 1, 'per_page': 1}, conf)
    param_query({'q': '*x', 'sort_by': 'downloads', 'order': 'desc', 'per_page': 1}, conf,
                setPageOutOfBounds=True)

    # Clears the downlaods
    pathstr = "(" + ("%s" * len(conf.paths)) + ")"
    conf.db.execute(("UPDATE movies SET downloads=0 WHERE path IN (%s)" % pathstr), *conf.paths.values())
    for tableKey in conf.paths.iterkeys():
        for i in range(len(conf.movies[tableKey])):
            conf.movies[tableKey][i]['downloads'] = 0
