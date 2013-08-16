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
    paths = {'movies': os.path.join(testdir, 'moviedir'), 'another': os.path.join(testdir, 'moviedir/anotherdir')}
    # Movies includes all the files and directories for each path,
    # filtering out dotfiles/dotdirectories and symlinks
    movies = {}
    for tablekey, path in paths.iteritems():
        namelist = []
        for dirpath, _, files in os.walk(path):
            reldir = os.path.relpath(dirpath, path)
            if reldir == '.' or reldir[0] != '.':
                namelist.append(torndb.Row({'name': reldir, 'downloads': 0}))
            else:
                # Skips the directory if it's a bad one
                continue
            for f in files:
                abspath = os.path.join(dirpath, f)
                if f[0] != '.' and not os.path.islink(abspath):
                    namelist.append(torndb.Row({'name': os.path.relpath(abspath, path), 'downloads': 0}))
        movies[tablekey] = namelist
    port = 10000
    db = torndb.Connection('127.0.0.1', 'movieserver', user="root")
    conf = torndb.Row({
        'srcpath': srcpath,
        'paths': paths,
        'movies': movies,
        'port': port,
        'serveraddress': 'http://localhost:' + str(port),
        'db': db,
        'handlers': torndb.Row({'main': '/main/', 'login': '/', 'checkAccess': '/checkAccess/', 'table': {},
                                'movie': {}})
    })
    for tableKey in paths.iterkeys():
        conf['handlers']['table'][tableKey] = '/main/table/%s/' % tableKey
        conf['handlers']['movie'][tableKey] = '/main/movie/%s/' % tableKey

    print 'Starting server on', conf.serveraddress
    proc = subprocess.Popen(['movieserver',
                             '-v', '2',
                             '-log_dir', testdir + '/logs',
                             '-src-path', conf['srcpath'],
                             '-path', 'movies=' + conf.paths['movies'],
                             '-path', 'another=' + conf.paths['another'],
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
