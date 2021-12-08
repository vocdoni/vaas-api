module go.vocdoni.io/api

go 1.16

require (
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/frankban/quicktest v1.13.0
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jmoiron/sqlx v1.2.1-0.20200615141059-0794cb1f47ee
	github.com/lib/pq v1.8.0
	github.com/prometheus/client_golang v1.10.0
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/shopspring/decimal v0.0.0-20200227202807-02e2044944cc // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	go.vocdoni.io/dvote v1.0.4-0.20211208183856-3cc820f5d0b2
	go.vocdoni.io/proto v1.13.3-0.20211202161242-17ea8a435be6
	golang.org/x/crypto v0.0.0-20210920023735-84f357641f63
	google.golang.org/protobuf v1.27.1
)

// Newer versions of the fuse module removed support for MacOS.
// Unfortunately, its downstream users don't handle this properly,
// so our builds simply break for GOOS=darwin.
// Until either upstream or downstream solve this properly,
// force a downgrade to the commit right before support was dropped.
// It's also possible to use downstream's -tags=nofuse, but that's manual.
// TODO(mvdan): remove once we've untangled module dep loops.
replace bazil.org/fuse => bazil.org/fuse v0.0.0-20200407214033-5883e5a4b512
