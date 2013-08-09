# Tests the movietable handler

import requests
import fnmatch

def test_handler_files(conf, conn):
    conn.execute("SELECT path, name FROM movies WHERE present=TRUE")
    rows = conn.fetchall()
    badpaths = [r['path'] for r in rows if r['path'] != conf['moviepath']]
    names = set([r['name'] for r in rows])

    assert len(badpaths) == 0
    assert names == conf['movies']

# Given the parameters (which aren't in string form yet), it runs
# a query and figures out the right asserts to make. Given a page
# argument, it will fetch all the pages up to and including that
# page and test those, not just the specified page
def param_query(params, conf):
    if 'q' in params:
        # The server runs a prefix match, so we add a star
        confmovies = set([f for f in conf['movies'] if fnmatch.fnmatch(f, params['q'] + '*')])
    else:
        confmovies = conf['movies']

    if 'per_page' in params:
        resp_len = min(params['per_page'], len(confmovies))
    else:
        resp_len = len(confmovies)

    strparams = {k: str(v) for k, v in params.iteritems()}

    if 'page' in params:
        assert 'per_page' in params
        allnames = set()
        for i in range(params['page']):
            strparams['page'] = str(i)
            req = requests.get(conf['serveraddress'] + conf['handlers']['movietable'], params=strparams)
            resp = req.json()

            assert resp[0]['total_entries'] == len(confmovies)
            assert len(resp[1]) == resp_len
            nameset = set([movie['Name'] for movie in resp[1]])
            assert nameset.issubset(confmovies)
            allnames.update(nameset)
        assert allnames.issubset(confmovies)
    else:
        assert 'per_page' not in params
        req = requests.get(conf['serveraddress'] + conf['handlers']['movietable'], params=strparams)
        resp = req.json()

        assert resp[0]['total_entries'] == len(confmovies)
        assert len(resp[1]) == resp_len
        nameset = set([movie['Name'] for movie in resp[1]])
        assert nameset == confmovies

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
    param_query({'q': '*x', 'page': 2, 'per_page': 2}, conf)

def test_qslashx_ptwo_pptwo(conf):
    param_query({'q': '*/x', 'page': 2, 'per_page': 2}, conf)

def test_out_of_bounds(conf):
    req = requests.get(conf['serveraddress'] + conf['handlers']['movietable'],
                       params={'q': '*x', 'page': '3', 'per_page': '1'})
    resp = req.json()

    confmovies = set([f for f in conf['movies'] if fnmatch.fnmatch(f, '*x*')])

    assert resp[0]['total_entries'] == len(confmovies)
    assert resp[0]['page'] == 1
    assert resp[0]['per_page'] == 1

    nameset = set([movie['Name'] for movie in resp[1]])
    assert nameset.issubset(confmovies)
