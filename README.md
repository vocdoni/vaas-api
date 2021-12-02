# Vocdoni Voting-as-a-Service API Server

[![GoDoc](https://godoc.org/go.vocdoni.io/api?status.svg)](https://godoc.org/go.vocdoni.io/api)
[![Go Report Card](https://goreportcard.com/badge/go.vocdoni.io/api)](https://goreportcard.com/report/go.vocdoni.io/api)
[![Coverage Status](https://coveralls.io/repos/github/vocdoni/vaas-api/badge.svg)](https://coveralls.io/github/vocdoni/vaas-api)

[![Join Discord](https://img.shields.io/badge/discord-join%20chat-blue.svg)](https://discord.gg/4hKeArDaU2)
[![Twitter Follow](https://img.shields.io/twitter/follow/vocdoni.svg?style=social&label=Follow)](https://twitter.com/vocdoni)

The Voting-as-a-Service API Server provides private REST API methods to integrators who want to use the Vocdoni voting protocol. Integrators can sign up for an account with a billing plan, and they will receive an authentication token. They can then use this token to create & manage organizations and allow organizations to then create voting processes. 

Note: this API is not intended to be used directly by organizations. The intended user is third-party who has their own site, application, or service, and wants to integrate voting into that service. Their users would only interact with their interface, which would handle all API calls. 

The manager is made of two components: a private database and a REST API. 

## Database

The VaaS database holds information about integrators, organizations, processes, etc, in order to easily provide this information to the REST API. 

### Design
A relational database is being used to store the necessary information. The following schema describes the involved relational entities:

![](misc/db/DBSchema.png)

The main entities are:
- `Integrator`: A third-party integrator of the VaaS API, including a billing plan and a set of organizations (customers of theirs)
- `Organization`: An organization identified by its entityID
- `Election`: A voting process belonging to a specific organization
- `Census`: A census for a voting process, containing a number of census items
- `CensusItem`: An item containing a public key corresponding to an eligible voter
- `BillingPlan`: A configuration item specifying the maximum census size and process count available to a given integrator's account

### Implementation
The database is designed as a relational DB, and is implemented in Postgres. Nevertheless, the DB calls are abastracted by an the interface `database/database.go`, allowing for other implementations as well.

For the performing the with Postgres queries we use [jmoiron/sqlx](github.com/jmoiron/sqlx), which uses the [lib/pq](github.com/lib/pq) module for connection.

Database migrations ara handled with the [rubenv/sql-migrate](github.com/rubenv/sql-migrate) module.


## APIs

### API Service

The API service, called `UrlAPI` in the codebase, contains the logic and components for the VaaS API.

The API service wraps:
- `router`: Manages the incoming requests
- `api`: Contains the authentication middleware
- `metrics agent`: Graphana and Prometheus metrics system
- `db`: The VaaS database
- `vocClient`: A client to make requests to the Vocdoni-Node gateways (communication with the Vochain)

#### REST API
The REST API includes the following endpoints:
- `Admin` calls for administrators (Vocdoni) to manage the set of Integrators and billing plans
- `Private` calls for integrators to manage organizations & voting processes
- `Quota` rate-limited public calls for end-users to submit votes & query voting process information

Available by default under `/api`.
A detailed version of the API can be found [here](/urlapi/README.md).


### Run
TDB
#### Running with docker
TDB
#### Tests

```bash
$ go test ./...
```
<p>&nbsp;</p>