syntax:
	find . -name '*.go' -print0 | xargs -0 gofmt -w -l *.go
	flake8 tests

test:
	py.test tests
