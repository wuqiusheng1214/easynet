module easynet

go 1.16

require (
	easyutil v0.0.0
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/websocket v1.5.0
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.19.0 // indirect
	github.com/smartwalle/dbs v1.2.0
)
replace (
	easyutil v0.0.0 => ../easyutil
)
