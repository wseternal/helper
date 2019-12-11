module github.com/wseternal/helper

go 1.13

replace github.com/wseternal/gorocksdb => ../gorocksdb

require (
	github.com/boltdb/bolt v1.3.1
	github.com/facebookgo/ensure v0.0.0-20160127193407-b4ab57deab51 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20150612182917-8dac2c3c4870 // indirect
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/go-sql-driver/mysql v1.4.1
	github.com/hashicorp/golang-lru v0.5.3
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/stretchr/testify v1.4.0 // indirect
	github.com/wseternal/gorocksdb v0.0.0-20190617101424-36bbc805f4f7
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f // indirect
	google.golang.org/appengine v1.6.1 // indirect
)
