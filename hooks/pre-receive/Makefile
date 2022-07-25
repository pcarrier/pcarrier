.PHONY: publish deploy

publish: dist/pre-receive
	rm -fr dist/.git && \
	cd dist && \
	git init && \
	git add pre-receive && \
	git commit -aSm build && \
	(git fetch https://pcarrier.com +refs/!hooks/pre-receive:refs/heads/existing && git merge --allow-unrelated-histories refs/heads/existing -m deploy --sign || echo no existing) && \
	git push https://pcarrier.com +refs/heads/main:refs/!hooks/pre-receive

deploy: dist/pre-receive
	scp dist/pre-receive pcarrier.com:/git/hooks

dist/pre-receive: main.go
	GOOS=linux GOARCH=amd64 go build -o dist/pre-receive .
