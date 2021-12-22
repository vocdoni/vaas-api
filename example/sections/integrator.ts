import { ProcessKeys } from "@vocdoni/voting"
import "isomorphic-unfetch"
import { getConfig } from "../util/config"

const config = getConfig()

export function wait(seconds: number) {
  return new Promise(resolve => {
    console.log("Waiting", seconds, "s")

    setTimeout(resolve, seconds * 1000)
  })
}

// INTEGRATOR

export async function createOrganization(name: string, apiKey: string) {
  const url = config.apiUrlPrefix + "/v1/priv/account/organizations"

  const body = {
    name,
    description: "Test organization",
    header: "https://my/header.jpeg",
    avatar: "https://my/avatar.png"
  }

  console.log("POST", url, body)

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + apiKey,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body),
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { organizationId, apiToken, txHash } = responseBody

  while (!(await getTransactionStatus(txHash, apiKey))) {
    await wait(15)
  }

  console.log("Created organization with ID", organizationId, "and API token", apiToken)
  return { organizationId, apiToken }
}

export async function getOrganizationPriv(id: string, apiKey: string) {
  const url = config.apiUrlPrefix + "/v1/priv/account/organizations/" + id

  const response = await fetch(url, {
    headers: {
      "Authorization": "Bearer " + apiKey,
    },
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()
  console.log("GET", url, responseBody)

  const { error } = responseBody
  if (error) throw new Error(error)

  const { apiToken, name, description, header, avatar } = responseBody

  console.log("Read organization with ID", id, ":", responseBody)
  return { apiToken, name, description, header, avatar }
}

export async function deleteOrganization(id: string, apiKey: string) {
  const url = config.apiUrlPrefix + "/v1/priv/account/organizations/" + id
  console.log("DELETE", url)

  const response = await fetch(url, {
    method: "DELETE",
    headers: {
      "Authorization": "Bearer " + apiKey
    },
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()
  const { error } = responseBody

  if (error) throw new Error(error)

  console.log("Deleted organization", id)
}

// ORGANIZATION - INTEGRATOR

export async function setOrganizationMetadata(id: string, apiKey: string) {
  const url = config.apiUrlPrefix + "/v1/priv/organizations/" + id + "/metadata"

  const body = {
    name: "Organization name 2",
    description: "Test organization bis",
    header: "https://my/header-2.jpeg",
    avatar: "https://my/avatar-2.png"
  }

  const response = await fetch(url, {
    method: "PUT",
    headers: {
      "Authorization": "Bearer " + apiKey,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body),
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { apiToken, name, description, header, avatar , txHash } = responseBody

  while (!(await getTransactionStatus(txHash, apiKey))) {
    await wait(15)
  }

  console.log("Updated organization with ID", id)
  return { apiToken, name, description, header, avatar }
}

export async function createSignedElection(organizationId: string, hiddenResults: boolean, apiKey: string) {
  const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/elections/signed"

  const startDate = new Date(Date.now() + 1000 * 60) // start time should be at least one minute from 'now()'
  const endDate = new Date(Date.now() + 1000 * 60 * 60)

  const body = {
    title: "Signed election",
    description: "Description here",
    header: "https://my/header.jpeg",
    streamUri: "https://youtu.be/1234",
    // startDate: startDate.toJSON(), //  "2021-12-10T11:20:53.769Z", // can be empty to start immediately when created
    endDate: endDate.toJSON(), //  "2021-12-15T12:00:00.000Z",
    questions: [
      {
        title: "Question 1 goes here",
        description: "(optional)",
        choices: ["Yes", "No", "Maybe"]
      },
      {
        title: "Question 2 title goes here",
        description: "(optional)",
        choices: ["Yes", "No", "Maybe", "Blank"]
      },
    ],
    confidential: false,  // Metadata access restricted to only census members
    hiddenResults, // Encrypt results until the process ends
    census: ""     // Empty when using a custom CSP
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + apiKey,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body),
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { electionId, txHash } = responseBody

  while (!(await getTransactionStatus(txHash, apiKey))) {
    await wait(15)
  }

  console.log("Created election with ID", electionId)
  return { electionId }
}

export async function createAnonymousElection(organizationId: string, hiddenResults: boolean, apiKey: string) {
  const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/elections/blind"

  const startDate = new Date(Date.now() + 1000 * 60) // start time should be at least one minute from 'now()'
  const endDate = new Date(Date.now() + 1000 * 60 * 60)

  const body = {
    title: "Anonymous election",
    description: "Description here",
    header: "https://my/header.jpeg",
    streamUri: "https://youtu.be/1234",
    startDate: startDate.toJSON(), //  "2021-12-10T11:20:53.769Z", // can be empty can be empty to start immediately when created
    endDate: endDate.toJSON(), //  "2021-12-15T12:00:00.000Z",
    questions: [
      {
        title: "Question 1 goes here",
        description: "(optional)",
        choices: ["Yes", "No", "Maybe"]
      },
      {
        title: "Question 2 title goes here",
        description: "(optional)",
        choices: ["Yes", "No", "Maybe", "Blank"]
      },
    ],
    confidential: false,  // Metadata access restricted to only census members
    hiddenResults, // Encrypt results until the process ends
    census: ""     // Empty when using a custom CSP
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + apiKey,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body),
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { electionId , txHash } = responseBody

  while (!(await getTransactionStatus(txHash, apiKey))) {
    await wait(15)
  }

  console.log("Created anonymous election", electionId)
  return { electionId }
}

type ElectionSummary = {
  title: string
  description: string
  header: string
  status: string
  startDate: string
  endDate: string
}
export async function listElectionsPriv(organizationId: string, apiKey: string): Promise<Array<ElectionSummary>> {
  const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/elections/signed"
  // const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/elections/blind"
  // const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/elections/active"
  // const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/elections/ended"
  // const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/elections/ended"

  const response = await fetch(url, {
    headers: {
      "Authorization": "Bearer " + apiKey,
    },
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  console.log("List organization elections", organizationId, ":", responseBody)
  return responseBody
}

type ElectionDetails = {
  type: string
  title: string
  description: string
  header: string
  streamUri: string
  questions: {
    title: string,
    description: string,
    choices: string[]
  }[],
  confidential: boolean,
  hiddenResults: boolean,
  census: string,
  status: string,
  resultsAggregation: string;
  resultsDisplay: string;
  endDate: Date;
  startDate: Date;
  encryptionPubKeys: {
    idx: number;
    key: string;
  }[]
}
export async function getElectionPriv(electionId: string, apiKey: string): Promise<ElectionDetails> {
  const url = config.apiUrlPrefix + "/v1/priv/elections/" + electionId

  const response = await fetch(url, {
    headers: {
      "Authorization": "Bearer " + apiKey,
    },
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  responseBody.startDate = responseBody.startDate ? new Date(responseBody.startDate) : null
  responseBody.endDate = new Date(responseBody.endDate)

  console.log("Get election", electionId, ":", responseBody)
  return responseBody
}

export async function getTransactionStatus(txHash: string, apiKey: string): Promise<boolean> {
  const url = config.apiUrlPrefix + "/v1/priv/transactions/" + txHash

  const response = await fetch(url, {
    headers: {
      "Authorization": "Bearer " + apiKey,
    },
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  let { mined } = responseBody
  mined = ((typeof mined) === "undefined") ? false : mined

  console.log("Get transaction", txHash, ":", mined)
  return mined
}
