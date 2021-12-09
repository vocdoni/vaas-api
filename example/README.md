Voting as a Service client example
---

Download and install [NodeJS](https://nodejs.org/en/) for your operating system.

Install the dependencies:

```sh
cd example
npm install
```

Run the example end-to-end flow:

```sh
npm start
```

## Sections

### Superadmin

These are the HTTP calls expected to be made by the admin running the service. They serve the purpose of managing the integrator accounts.

### Integrator

The calls on this section are meant to be executed from the backend of the Integrator. They allow to:
- Create and manage organizations that can conduct elections
- Create and manage elections
- Create and manage censuses (optional)

### Voter

The calls on this section belong to the public API and are meant to be executed by voters from their web browser. These allow to:
- Get the details of an election/organization
- Submit a ballot
- See the results
- Check the status of a ballot
