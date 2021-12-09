import "isomorphic-unfetch"
import { getConfig } from "../util/config"

const config = getConfig()

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

  const { organizationId, apiToken } = responseBody

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

  const { apiToken, name, description, header, avatar } = responseBody

  console.log("Updated organization with ID", id)
  return { apiToken, name, description, header, avatar }
}

export async function createSignedElection(organizationId: string, apiKey: string) {
  const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/processes/signed"

  const body = {
    title: "Signed election",
    description: "Description here",
    header: "https://my/header.jpeg",
    streamUri: "https://youtu.be/1234",
    startDate: "2021-10-25T11:20:53.769Z", // can be empty
    endDate: "2021-10-30T12:00:00.000Z",
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
    confidential: true,  // Metadata access restricted to only census members
    hiddenResults: true, // Encrypt results until the process ends
    census: ""     // Empty when using a custom CSP
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

  const { processId: electionId } = responseBody

  console.log("Updated organization with ID", organizationId)
  return { electionId }
}

export async function createAnonymousElection(organizationId: string, apiKey: string) {
  const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/processes/signed"

  const body = {
    title: "Signed election",
    description: "Description here",
    header: "https://my/header.jpeg",
    streamUri: "https://youtu.be/1234",
    startDate: "2021-10-25T11:20:53.769Z", // can be empty
    endDate: "2021-10-30T12:00:00.000Z",
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
    confidential: true,  // Metadata access restricted to only census members
    hiddenResults: true, // Encrypt results until the process ends
    census: ""     // Empty when using a custom CSP
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

  const { processId: electionId } = responseBody

  console.log("Updated organization with ID", organizationId)
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
  const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/processes/signed"
  // const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/processes/blind"
  // const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/processes/active"
  // const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/processes/ended"
  // const url = config.apiUrlPrefix + "/v1/priv/organizations/" + organizationId + "/processes/ended"

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
    title: "Question 1",
    description: "(optional)",
    choices: ["Yes", "No", "Maybe"]
  }[],
  confidential: true,
  hiddenResults: true,
  census: string
  status: string
}
export async function getElectionPriv(electionId: string, apiKey: string): Promise<ElectionDetails> {
  const url = config.apiUrlPrefix + "/v1/priv/processes/" + electionId

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

  console.log("Get election", electionId, ":", responseBody)
  return responseBody
}
