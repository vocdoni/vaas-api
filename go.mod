module go.vocdoni.io/api

go 1.16

require (
	github.com/766b/chi-prometheus v0.0.0-20211217152057-87afa9aa2ca8 // indirect
	github.com/DataDog/zstd v1.4.8 // indirect
	github.com/adlio/schema v1.2.3 // indirect
	github.com/arnaucube/go-blindsecp256k1 v0.0.0-20211204171003-644e7408753f
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/cockroachdb/errors v1.8.9 // indirect
	github.com/cockroachdb/pebble v0.0.0-20220224015757-894b57aa32be // indirect
	github.com/dgraph-io/badger/v2 v2.2007.4 // indirect
	github.com/dgraph-io/badger/v3 v3.2103.2 // indirect
	github.com/ethereum/go-ethereum v1.10.16
	github.com/frankban/quicktest v1.14.2
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/uuid v1.3.0
	github.com/hashicorp/hcl v1.0.1-vault-3 // indirect
	github.com/iden3/go-iden3-crypto v0.0.13 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jmoiron/sqlx v1.2.1-0.20200615141059-0794cb1f47ee
	github.com/klauspost/cpuid/v2 v2.0.11 // indirect
	github.com/lib/pq v1.10.4
	github.com/libp2p/go-libp2p-gostream v0.3.1 // indirect
	github.com/libp2p/go-libp2p-http v0.2.1 // indirect
	github.com/libp2p/go-libp2p-noise v0.2.2 // indirect
	github.com/libp2p/go-libp2p-peerstore v0.2.10 // indirect
	github.com/libp2p/go-libp2p-swarm v0.5.3 // indirect
	github.com/libp2p/go-tcp-transport v0.2.8 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/marten-seemann/qtls-go1-17 v0.1.0 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/dns v1.1.46 // indirect
	github.com/mimoo/StrobeGo v0.0.0-20220103164710-9a04d6ca976b // indirect
	github.com/multiformats/go-base32 v0.0.4 // indirect
	github.com/multiformats/go-multicodec v0.3.0 // indirect
	github.com/multiformats/go-multihash v0.1.0 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.17.0 // indirect
	github.com/petermattis/goid v0.0.0-20220111183729-e033e1e0bdb5 // indirect
	github.com/pressly/goose/v3 v3.3.1 // indirect
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/statsd_exporter v0.22.4 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/shopspring/decimal v0.0.0-20200227202807-02e2044944cc // indirect
	github.com/spf13/afero v1.8.1 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/tendermint/tm-db v0.6.7 // indirect
	github.com/timshannon/badgerhold/v3 v3.0.0 // indirect
	github.com/vocdoni/arbo v0.0.0-20220204101222-688a2e814db0 // indirect
	github.com/vocdoni/blind-csp v0.1.5-0.20220214165159-4620baa07fa4
	github.com/whyrusleeping/cbor-gen v0.0.0-20220223114253-ebcc1e8ce85b // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.vocdoni.io/dvote v1.0.4-0.20220310171340-ae10a77cde5b
	go.vocdoni.io/proto v1.13.3-0.20220203130255-cbdb9679ec7c
	go4.org v0.0.0-20201209231011-d4a079459e60 // indirect
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292
	golang.org/x/exp v0.0.0-20220218215828-6cf2b201936e // indirect
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b // indirect
	golang.org/x/sys v0.0.0-20220224120231-95c6836cb0e7 // indirect
	golang.org/x/tools v0.1.9 // indirect
	google.golang.org/genproto v0.0.0-20220222213610-43724f9ea8cf // indirect
	google.golang.org/protobuf v1.27.1
	gopkg.in/ini.v1 v1.66.4 // indirect
	lukechampine.com/blake3 v1.1.7 // indirect
)

// Newer versions of the fuse module removed support for MacOS.
// Unfortunately, its downstream users don't handle this properly,
// so our builds simply break for GOOS=darwin.
// Until either upstream or downstream solve this properly,
// force a downgrade to the commit right before support was dropped.
// It's also possible to use downstream's -tags=nofuse, but that's manual.
// TODO(mvdan): remove once we've untangled module dep loops.
replace bazil.org/fuse => bazil.org/fuse v0.0.0-20200407214033-5883e5a4b512
