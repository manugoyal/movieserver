syntax:
	find . -name '*.go' -print0 | xargs -0 gofmt -w -l *.go
	. venv/bin/activate; flake8 tests

test:
	. venv/bin/activate; py.test tests $(options)

testdeps:
	command -v virtualenv >/dev/null 2>&1 || { echo >&2 "movieserver requires virtualenv, but it's not installed: aborting."; exit 1; }
	command -v mysql_config >/dev/null 2>&1 || { echo >&2 "movieserver requires mysql_config but it's not installed: aborting."; exit 1; }
	virtualenv venv
	. venv/bin/activate; pip install -r conf/requirements.txt
