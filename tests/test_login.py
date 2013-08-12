# Tests the checkAccess handler

import requests

def test_good_login(conf):
    user, password = ('success', 'success')
    conf.db.execute("REPLACE INTO login VALUES (%s, %s)", user, password)

    goodLogin = requests.post(conf.serveraddress + conf.handlers.checkAccess,
                              data={'username': user, 'password': password})
    assert goodLogin.status_code == 200

    indexPage = requests.get(conf.serveraddress + conf.handlers.main)
    assert indexPage.text == goodLogin.text

    conf.db.execute("DELETE FROM login WHERE user=%s and password=%s", user, password)

def test_bad_login(conf):
        badLogin = requests.post(conf.serveraddress + conf.handlers.checkAccess,
                                 data={'username': 'failure', 'password': 'failure'})
        assert badLogin.status_code == 403
