module github.com/wseternal/helper

go 1.13

require (
	github.com/boltdb/bolt v1.3.1
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/go-sql-driver/mysql v1.4.1
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/wseternal/gorocksdb v0.0.0-20190519120508-025c3cf4ffb4
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f // indirect
	google.golang.org/appengine v1.6.1 // indirect
)

replace github.com/wseternal/gorocksdb => ../gorocksdb
