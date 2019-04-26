module github.com/wseternal/helper

go 1.13

require (
	github.com/boltdb/bolt v1.3.1
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/go-sql-driver/mysql v1.4.1
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/tecbot/gorocksdb v0.0.0-20181010114359-8752a9433481
	golang.org/x/sys v0.0.0-20190422165155-953cdadca894 // indirect
	google.golang.org/appengine v1.5.0 // indirect
)

replace github.com/tecbot/gorocksdb => ../../tecbot/gorocksdb
