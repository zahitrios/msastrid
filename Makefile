.PHONY: build clean deploy

deploy-prod: clean
	make STAGE=prod replace-environment
	make build
	make manual-deploy
	make restore-environment

deploy-stg: clean
	make STAGE=stg replace-environment
	make build
	make manual-deploy
	make restore-environment

run: clean build
	./bin/products

manual-deploy:
	npm i
	sls deploy --force --verbose

replace-environment:
	cat .env.${STAGE} > .env
	mv ./products/local.env ./products/local.env.backup || true
	cat .env.${STAGE} > ./products/local.env
	sed -i "/stage/c\  stage: ${STAGE}" ./serverless.yml

restore-environment:
	rm .env
	rm ./products/local.env
	mv ./products/local.env.backup ./products/local.env || true
	sed -i "/stage/c\  stage: stg-hello" ./serverless.yml

build:
	go mod download
	env GOARCH=amd64 GOOS=linux go build -mod=mod -ldflags="-s -w" -o bin/products products/main.go

clean:
	rm -rf ./bin