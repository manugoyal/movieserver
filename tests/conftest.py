import pytest
import MySQLdb
import MySQLdb.cursors
import os.path
import inspect
import subprocess
import time
import signal
import torndb

# Sets up the server on port 10000 and also a database connection
@pytest.fixture(scope="session")
def conf(request):
    testdir = os.path.dirname(os.path.abspath(inspect.getfile(inspect.currentframe())))
    srcpath = os.path.abspath(testdir + '/..')
    moviepath = testdir + '/moviedir'
    movies = [torndb.Row({'name': (dirpath + "/" + f)[len(moviepath) + 1:], 'downloads': 0})
                              for dirpath, _, files in os.walk(moviepath)
                              for f in files if len(f) > 0 and f[0] != '.']
    port = 10000
    db = torndb.Connection('127.0.0.1', 'movieserver', user="root")
    conf = torndb.Row({
        'srcpath': srcpath,
        'moviepath': moviepath,
        'movies': movies,
        'port': port,
        'serveraddress': 'http://localhost:' + str(port),
        'db': db,
        'handlers': torndb.Row({'main': '/main/', 'movietable': '/main/table/movie',
                                'movie': '/main/movie/', 'login': '/',
                                'checkAccess': '/checkAccess/'})
    })

    print 'Starting server on', conf.serveraddress
    proc = subprocess.Popen(['movieserver',
                            '-v', '2',
                            '-log_dir', testdir + '/logs',
                            '-src-path', conf['srcpath'],
                            '-movie-path', conf['moviepath'],
                            '-port', str(port)])
    conf.proc = proc
    time.sleep(5)

    def teardown():
        db.close()
        proc.send_signal(signal.SIGINT)
        proc.wait()

    signal.signal(signal.SIGINT, teardown)

    request.addfinalizer(teardown)
    return conf
