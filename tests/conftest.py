import pytest
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
    # Movies includes all the non-dot directories and files plus the
    # moviedir directory itself, as a '.'
    movies = [torndb.Row({'name': os.path.join(dirpath, item)[len(moviepath) + 1:], 'downloads': 0})
              for dirpath, dirnames, files in os.walk(moviepath)
              for item in files + dirnames if len(item) > 0 and item[0] != '.']
    movies += [torndb.Row({'name': '.', 'downloads': 0})]
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
