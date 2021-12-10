# VaaS API Specification

The service exposes an HTTP Restful API with the following endpoints. 

## Internal API
The group of calls below is intended for the admin running the service itself. 

**Superadmin related**

---

### Create an integrator account
<details>
<summary>Example</summary>

#### Request 
```bash
curl -X POST -H "Bearer: <superadmin-key>" https://server/v1/admin/accounts
```

#### Request body
```json
{
	"cspUrlPrefix": "my-csp-url-prefix",
	"cspPubKey": "hexBytes",
	"name": "My integrator account",
}
```

#### HTTP 200
```json
{
	"id": "1234567890",
	"apiKey": "ksjdhfksjdh..."   // The integrator (secret) key to use the API
}
```

#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>
    
### Update an integrator account
<details>
<summary>Example</summary>

#### Request 
```bash
curl -X PUT -H "Bearer: <superadmin-key>" https://server/v1/admin/accounts/<id>
```

#### Request body
```json
{
    "name": "My integrator account",
    "plan": "billing-plan-key"
}
```

#### HTTP 200
```json
{
    "id": "1234567890"
}
```

#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>
    
### Reset an integrator API key  
<details>
<summary>Example</summary> 

#### Request 
```bash
curl -X PATCH -H "Bearer: <superadmin-key>" https://server/v1/admin/accounts/<id>/key
```

#### HTTP 200
```json
{
    "id": "1234567890",
    "apiKey": "zxcvbnm"
}

```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```

### Get an integrator account
<details>
<summary>Example</summary>

#### Request 
```bash
curl -H "Bearer: <superadmin-key>" https://server/v1/admin/accounts/<id>

```
#### HTTP 200
```json
{
    "name": "My integrator account",
    "plan": "billing-plan-key"
}

```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>
</details>

### Delete an integrator account
<details>
<summary>Example</summary>

#### Request 
```bash
curl -X DELETE -H "Bearer: <superadmin-key>" https://server/v1/admin/accounts/<id>
```
#### HTTP 200
```json
// empty response
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>
</details>

## Integrator API (Private)

**Integrator related**

---

The following endpoints are authenticated by using the integrator secret key. They allow integrators to manage the organizations related to their customers. 

### Create an organization
<details>
<summary>Example</summary>

#### Request 
```bash
curl -X POST -H "Bearer: <integrator-key>" https://server/v1/priv/account/organizations
```

#### Request body
```json
{
    "name": "Organization name",
    "description": "",
    "header": "https://my/header.jpeg",
    "avatar": "https://my/avatar.png"
}
```
#### HTTP 200
```json
{
    "organizationId": "0x1234...",
    "apiToken": "qwertyui...",   // API token for public voting endpoints
    "apiKey": "asdfghjk..."      // Secret API key to manage the organization
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>
</details>

### Get an organization
<details>
<summary>Example</summary>

#### Request 
```bash
curl -H "Bearer: <integrator-key>" https://server/v1/priv/account/organizations/<organizationId>
```

#### HTTP 200
```json
{
    "apiToken": "qoiuwhgoiauhsdaiouh",   // the public API token
    "name": "Organization name",
    "description": "",
    "header": "https://my/header.jpeg",
    "avatar": "https://my/avatar.png"
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Remove an organization
<details>
<summary>Example</summary>

#### Request 
```bash
curl -X DELETE -H "Bearer: <integrator-key>" https://server/v1/priv/account/organizations/<organizationId>
```

#### HTTP 200
```json
// empty response
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Reset the public API key of an organization
<details>
<summary>Example</summary>

#### Request 
```bash
curl -X PATCH -H "Bearer: <integrator-key>" https://server/v1/account/organizations/<id>/key
```

#### HTTP 200
```json
{
    "id": "1234567890",
    "apiKey": "zxcvbnm"
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

**Organization related**

---

These methods are also intended for integrators, but they are expected to do the duties of an organization managing a proposal.

### Set Organization metadata
<details>
<summary>Example</summary>

#### Request 
```bash
curl -X PUT -H "Bearer: <integrator-key>" https://server/v1/priv/organizations/<organizationId>/metadata
```

#### Request body
```json
{
    "name": "Organization name",
    "description": "",
    "header": "https://my/header.jpeg",
    "avatar": "https://my/avatar.png"
}
```
#### HTTP 200
```json
{
    "organizationId": "0x1234..."
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Create a process
Generates a Merkle Tree with the given current census keys and generates a voting process with the given metadata. 
<details>
<summary>Example</summary>


#### Request 
```bash
curl -X POST -H "Bearer: <integrator-key>" https://server/v1/priv/organizations/<organizationId>/elections/signed
curl -X POST -H "Bearer: <integrator-key>" https://server/v1/priv/organizations/<organizationId>/elections/blind
```

#### Request body
```json
{
    "title": "Important election",
    "description": "Description here",
    "header": "https://my/header.jpeg",
    "streamUri": "https://youtu.be/1234",
    "startDate": "2021-10-25T11:20:53.769Z", // can be empty
    "endDate": "2021-10-30T12:00:00.000Z",
    "questions": [
        {
            "title": "Question 1",
            "description": "(optional)",
            "choices": ["Yes", "No", "Maybe"]  // simplified version of title/value
        }, {...}
    ],
    "confidential": true,  // Metadata access restricted to only census members
    "hiddenResults": true, // Encrypt results until the process ends
    "census": "<censusId>"
}
```

#### HTTP 200
```json
{
    "processId": "0x1234..."
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### List processes (filtered)
Allows unrestricted listing, paging and filtering for the integrator backend to display all info to organization admin's.
<details>
<summary>Example</summary>


#### Request 
```bash
curl -H "Bearer: <integrator-key>" https://server/v1/priv/organizations/<organizationId>/elections/signed
curl -H "Bearer: <integrator-key>" https://server/v1/priv/organizations/<organizationId>/elections/blind

curl -H "Bearer: <integrator-key>" https://server/v1/priv/organizations/<organizationId>/elections/active
curl -H "Bearer: <integrator-key>" https://server/v1/priv/organizations/<organizationId>/elections/ended
curl -H "Bearer: <integrator-key>" https://server/v1/priv/organizations/<organizationId>/elections/upcoming
```
#### HTTP 200
```json
[
    {
        "title": "Important election",
        "description": "",
        "header": "https://my/header.jpeg",
        "status": "READY",
        "startDate": "2021-10-25T11:20:53.769Z", // can be empty
        "endDate": "2021-10-30T12:00:00.000Z",
    }, {...}
]
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Get a process
Allows unrestricted access for the integrator backend to display all info to organization admin's.
Confidential processes do not require any additional step, just the integrator API key.
<details>
<summary>Example</summary>


#### Request 
```bash
curl -H "Bearer: <integrator-key>" https://server/v1/priv/elections/<processId>
```

#### Request body
```json
{
    "type": "signed", // blind, ...
    "title": "Important election",
    "description": "Description goes here",
    "header": "https://my/header.jpeg",
    "streamUri": "https://youtu.be/1234",
    "questions": [
        {
            "title": "Question 1",
            "description": "(optional)",
            "choices": ["Yes", "No", "Maybe"]
        }, {...}
    ],
    "confidential": true,  // Metadata access restricted to only census members
    "hiddenResults": true, // Encrypt results until the process ends
    "census": "<censusId>",
    "status": "PAUSED"
}
```
#### HTTP 200
```json
{
    "processId": "0x1234..."
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Create a census
A census where public keys or token slots (that will eventually contain a public key) are stored. A census can start with 0 items, and public keys can be imported later on.

If census tokens are allocated, users will need to generate a wallet on the frontend and register the public key by themselves. This prevents both the API and the integrator from gaining access to the private key.

<details>
<summary>Example</summary>


#### Request 
```bash
curl -X POST -H "Bearer: <integrator-key>" https://server/v1/priv/censuses
```

#### Request body
```json
{
    "name": "2021 members"
}
```
#### HTTP 200
```json
{
    "censusId": "123456789..."
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Add N census tokens (once)
Creates N census tokens for voters to register their public key in the future. 

<details>
<summary>Example</summary>


#### Request 
```bash
curl -X POST -H "Bearer: <integrator-key>" https://server/v1/priv/censuses/<censusId>/tokens/flat
```

#### Request body
```json
{
    "size": 450
}
```
#### HTTP 200
```json
{
    "censusId": "123456789...",
    "size": 700,  // new size
    "tokens": [
        "jashdlkfjahs", "uyroeituwyert", "e7rg9e87rn9", ... // x 450
    ]
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Add weighted census tokens (once)
Creates weighted census tokens so that voters with the token can register their public key to the appropriate census weight. 

<details>
<summary>Example</summary>


#### Request 
```bash
curl -X POST -H "Bearer: <integrator-key>" https://server/v1/priv/censuses/<censusId>/tokens/weighted
```

#### Request body
```json
{
    "weights": [
        40, 70, 200
    ]
}
```
#### HTTP 200
```json
{
    "censusId": "123456789...",
    "size": 700,  // new size
    "tokens": [
        { "token": "jashdlkfjahs", weight: 40 },
        { "token": "uyroeituwyert", weight: 70 },
        { "token": "e7rg9e87rn9", weight: 200 }
    ]
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Get census token (see a registered public key)

Allows integrators to check the status of a census token, given to a user.

If the token has already been redeemed, the public key will be used as part of the census normally.

<details>
<summary>Example</summary>

#### Request 
```bash
curl -H "Bearer: <integrator-key>" https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
```

#### HTTP 200
```json
{
    "publicKey": "",   // no public key yet
    "weight": 20
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Remove a census token or public key from a census
Removes the given token or key from the given census. The next time it is used, the new key will be in effect.

<details>
<summary>Example</summary>


#### Request 
```bash
curl -X DELETE -H "Bearer: <integrator-key>" https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
curl -X DELETE -H "Bearer: <integrator-key>" https://server/v1/priv/censuses/<censusId>/keys/<publicKey>
```

#### HTTP 200
```json
// empty response
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Import public keys into a census
Import a group of public keys to an existing census. All voters have the same weight (1).

<details>
<summary>Example</summary>


#### Request 
```bash
curl -X POST -H "Bearer: <integrator-key>" https://server/v1/priv/censuses/<censusId>/import/flat
```

#### Request body
```json
{
    "publicKeys": [
        "0x0312345678...",
        "0x0223456789...",
        ...
    ]
}
```
#### HTTP 200
```json
{
    "censusId": "123456789...",
    "size": 300
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Import weighted public keys into a census
Import a group of public keys to an existing census, using their respective weight. 

<details>
<summary>Example</summary>


#### Request 
```bash
curl -X POST -H "Bearer: <integrator-key>" https://server/v1/priv/censuses/<censusId>/import/weighted
```

#### Request body
```json
{
    "entries": [
        { "publicKey": "0x0312345678...", "weight": 100 },
        { "publicKey": "0x02234567890...", "weight": 150 },
        ...
    ]
}
```
#### HTTP 200
```json
{
    "censusId": "123456789...",
    "size": 300
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### End/start/pause/cancel a process
<details>
<summary>Example</summary>

#### Request 
```bash
curl -X PUT -H "Bearer: <integrator-key>" https://server/v1/priv/elections/<processId>/status
```

#### Request body
```json
{
    "status": "PAUSED" // READY, PAUSED, ENDED, CANCELED
}
```
#### HTTP 200
```json
{
    "censusId": "123456789..."
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>


## Public API
(token API authenticated, voter apps call it directly)

### Get Organization data
<details>
<summary>Example</summary>

#### Request 
```bash
curl -H "Bearer: <organization-api-token>" https://server/v1/pub/organizations/<organizationId>
```

#### HTTP 200
```json
{
    "name": "Organization name",
    "description": "",
    "header": "https://my/header.jpeg",
    "avatar": "https://my/avatar.png"
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Get process list (per organization)
<details>
<summary>Example</summary>

#### Request 
```bash
curl -H "Bearer: <manager-key>" https://server/v1/pub/organizations/<organizationId>/elections/active
curl -H "Bearer: <manager-key>" https://server/v1/pub/organizations/<organizationId>/elections/ended
curl -H "Bearer: <manager-key>" https://server/v1/pub/organizations/<organizationId>/elections/upcoming

```
#### HTTP 200
```json
[
    {
        "title": "Important election",
        "description": "",
        "header": "https://my/header.jpeg",
        "status": "READY",
        "startDate": "2021-10-25T11:20:53.769Z", // can be empty
        "endDate": "2021-10-30T12:00:00.000Z",
    }, {...}
]
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Get process info – non-confidential
<details>
<summary>Example</summary>

#### Request 
```bash
curl -H "Bearer: <organization-api-token>" https://server/v1/pub/elections/<processId>
```

#### HTTP 200
```json
{
    "type": "signed-plain", // blind-plain, ...
    "title": "Important election",
    "description": "Description goes here",
    "header": "https://my/header.jpeg",
    "streamUri": "https://youtu.be/1234",
    "questions": [
        {
            "title": "Question 1",
            "description": "(optional)",
            "choices": ["Yes", "No", "Maybe"]
        }, {...}
    ],
    "status": "READY",
    "voteCount": 1234,
    "results": [   // Empty array when no results []
        [ { "title": "Yes", "value": 1234, "title": "No", "value": 2345 } ],
        [ { "title": "Yes", "value": 22, "title": "No", "value": 33 } ]
    ]
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Get process info – confidential
Provides the details of a confidential voting process if the user holds a wallet that belongs to its census.

If `{processId, address}` has been signed by the CSP, then it gets the Vochain parameters, decrypts the metadata and returns it to the caller.

URL Params:

- process-id
- pid-signed: `sign(processId, voterPrivK)`
- csp-signature: `sign({processId, address}, cspPrivK)`
    - [See here](https://www.notion.so/Vocdoni-API-v1-86357bf911a24e33ab7159a2b6e54632)

<details>
<summary>Example</summary>

#### Request 
```bash
curl -H "Bearer: <entity-api-token>" https://server/v1/pub/elections/<process-id>/auth/<pid-signed>/<csp-signature>
```

#### HTTP 200
```json
{
    "type": "signed-plain", // blind-plain, ...
    "title": "Important election",
    "description": "Description goes here",
    "header": "https://my/header.jpeg",
    "streamUri": "https://youtu.be/1234",
    "questions": [
        {
            "title": "Question 1",
            "description": "(optional)",
            "choices": ["Yes", "No", "Maybe"]
        }, {...}
    ],
    "status": "READY",
    "voteCount": 1234,
    "results": [   // Empty array when no results []
        [ { "title": "Yes", "value": 1234, "title": "No", "value": 2345 } ],
        [ { "title": "Yes", "value": 22, "title": "No", "value": 33 } ]
    ]
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Requesting a census proof
People voting on a signed process will need to package a vote envelope using the result of this call. 

<details>
<summary>Example</summary>

#### Request 
```bash
curl -H "Bearer: <organization-api-token>" https://server/v1/pub/elections/<processId>/proof
```

#### Request body
```json
{
    "signature": "0x12345678..." // signing a well-known payload using the wallet
}
```
#### HTTP 200
```json
{
    "proof": "..."
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Submitting a vote (signed or blind)

Voters using the tiny JS SDK will get a base64 bundle including the vote and the census proof. This payload is submitted as a base64 string.

<details>
<summary>Example</summary>

#### Request 
```bash
curl -X POST -H "Bearer: <organization-api-token>" https://server/v1/pub/elections/<processId>/vote
```

#### Request body
```json
{
    "vote": "<base64-signed-vote-transaction>" // see dvote-js
}
```
#### HTTP 200
```json
{
    "nullifier": "0x12345678..."
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Getting a ballot (nullifier)
Voters can check the status of their vote here, and eventually check the explorer link, for independent confirmation.

<details>
<summary>Example</summary>


#### Request 
```bash
curl -H "Bearer: <organization-api-token>" https://server/v1/pub/nullifiers/<nullifier>
```
#### HTTP 200
```json
{
    "processId": "0x12345678...",
    "registered": true,
    "explorerUrl": "https://vaas.explorer.vote/nullifiers/0x12345678"
}
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>

### Registering a voter's public key

This process needs to be done by the integrator's frontend, once. 

As soon as a user runs an updated UI version, a new private key should be generated, and the public key should be registered, so that new votes can use this key.

If the wallet is lost, the integrator will need to remove the pubKey from the census and create a new census token when the new wallet is available. 

<details>
<summary>Example</summary>

#### Request 
```bash
curl -X POST -H "Bearer: <organization-api-token>" https://server/v1/pub/censuses/<censusId>/token
```

#### Request body
```json
{
    "censusToken": "jashdlkfjahs",
    "publicKey": "0x03abcdef0123456789..."
}
```
#### HTTP 200
```json
// empty response
```
#### HTTP 400
```json
{
    "error": "Message goes here"
}
```
</details>
