fmt:
	find . -name '*.go' -print0 | xargs -0 gofmt -w -l *.go
