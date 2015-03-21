# imagedb

A Web-based image viewer with tags and search

## Usage

At the moment you must run the binary in the same directory, after installing bower components.

    go get -d github.com/holizz/imagedb
    cd $GOPATH/src/github.com/holizz/imagedb
    bower install
    go build
    PORT=3000 MONGODB=localhost:27017 ./imagedb
