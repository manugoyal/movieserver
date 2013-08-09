# Tests the movie handler
import requests

def test_regular(conf):
    for movie in conf['movies']:
        req = requests.get(conf['serveraddress'] + conf['handlers']['movie'] + movie)
        filetext = open(conf['moviepath'] + '/' + movie).read()
        assert req.status_code == 200
        # It is encoded as a binary-stream, so req.content
        # contains the raw bytes
        assert filetext == req.content

def test_bogus(conf):
    req = requests.get(conf['serveraddress'] + conf['handlers']['movie'] + 'nonexistentfile.goober')
    assert req.status_code == 404
