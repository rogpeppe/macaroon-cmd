.PHONY: all install

all:
	GOARCH=amd64 go install ${ROOT}/macaroond ${ROOT}/macaroon
	GOARCH=amd64 go build -o ./bin/amd64/macaroond ${ROOT}/macaroond
	GOARCH=amd64 go build -o ./bin/amd64/macaroon ${ROOT}/macaroon

# note: the install target is used by snapcraft build
install: all
	test -d ${DESTDIR} || (echo DESTDIR not set; exit 1)
	cp -r ./bin ${DESTDIR}

ROOT=github.com/rogpeppe/macaroon-cmd/cmd
DAEMON=${ROOT}/macaroond
COMMAND=$ROOT}/macaroon
