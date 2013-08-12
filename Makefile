syntax:
	find . -name '*.go' -print0 | xargs -0 gofmt -w -l *.go
	. venv/bin/activate; flake8 tests

test:
	. venv/bin/activate; py.test tests $(options)

testdeps:
	virtualenv venv
	. venv/bin/activate; pip install -r conf/requirements.txt
