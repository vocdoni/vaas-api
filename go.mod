module go.vocdoni.io/api

go 1.16

require (
	github.com/arnaucube/go-blindsecp256k1 v0.0.0-20211204171003-644e7408753f
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/ethereum/go-ethereum v1.10.13
	github.com/frankban/quicktest v1.14.0
	github.com/google/uuid v1.3.0
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jmoiron/sqlx v1.2.1-0.20200615141059-0794cb1f47ee
	github.com/lib/pq v1.10.4
	github.com/prometheus/client_golang v1.12.0
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/shopspring/decimal v0.0.0-20200227202807-02e2044944cc // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/vocdoni/blind-csp v0.1.5-0.20220209203910-52b60a81fa7f
	go.vocdoni.io/dvote v1.0.4-0.20220208144419-fb9d208c920b
	go.vocdoni.io/proto v1.13.3-0.20220203130255-cbdb9679ec7c
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce
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
