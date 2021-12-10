import "isomorphic-unfetch"
import { getConfig } from "../util/config"

const config = getConfig()

//////////////////////////////////////////////////////////////////////////
// API endpoint calls
//////////////////////////////////////////////////////////////////////////

// VOTER

export async function getOrganizationPub(organizationId: string, orgApiToken: string) {
  const url = config.apiUrlPrefix + "/v1/pub/organizations/" + organizationId

  const response = await fetch(url, {
    headers: {
      "Authorization": "Bearer " + orgApiToken,
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

  const { name, description, header, avatar } = responseBody

  console.log("Read organization with ID", organizationId, ":", responseBody)
  return { name, description, header, avatar }
}


type ElectionSummary = {
  title: string,
  description: string,
  header: string,
  status: "READY" | "ENDED" | "CANCELED" | "PAUSED" | "RESULTS",
  startDate: string, // JSON date
  endDate: string // JSON date
}
export async function getElectionListPub(organizationId: string, status: "active" | "ended" | "upcoming", orgApiToken: string) {
  const url = config.apiUrlPrefix + "/v1/pub/organizations/" + organizationId + "/processes/" + status

  const response = await fetch(url, {
    headers: {
      "Authorization": "Bearer " + orgApiToken,
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

  console.log("Get organization election ID's", organizationId, ":", responseBody)
  return responseBody as ElectionSummary[]
}

type ElectionDetail = {
  type: "signed-plain" | "blind-plain"
  title: string
  description: string
  header: string
  streamUri: string
  questions: {
    title: string,
    description: string,
    choices: [string]
  }[],
  status: "READY" | "ENDED" | "CANCELED" | "PAUSED" | "RESULTS",
  voteCount: number,
  results: Array<Array<{ title: string, value: string }>> // Empty arrays when no results []
}
export async function getElectionInfoPub(electionId: string, orgApiToken: string) {
  const url = config.apiUrlPrefix + "/v1/pub/processes/" + electionId

  const response = await fetch(url, {
    headers: {
      "Authorization": "Bearer " + orgApiToken,
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
  return responseBody as ElectionDetail
}

export async function getElectionSecretInfoPub(electionId: string, cspSharedKey: string, orgApiToken: string) {
  const url = config.apiUrlPrefix + "/v1/pub/processes/" + electionId + "/auth/" + cspSharedKey

  const response = await fetch(url, {
    headers: {
      "Authorization": "Bearer " + orgApiToken,
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
  return responseBody as ElectionDetail
}

//////////////////////////////////////////////////////////////////////////
// CSP endpoint calls - Standard
//////////////////////////////////////////////////////////////////////////

export async function getElectionSharedKey(electionId: string, signedElectionId: string, orgApiToken: string): Promise<string> {
  const url = config.cspUrlPrefix + "/v1/auth/processes/" + electionId + "/sharedKey"

  const body = {
    "authData": [signedElectionId]
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + orgApiToken,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body)
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { sharedKey } = responseBody

  console.log("Get election shared key", sharedKey)
  return sharedKey
}

export async function getCspSigningTokenPlain(electionId: string, signedElectionId: string, orgApiToken: string): Promise<string> {
  const url = config.cspUrlPrefix + "/v1/auth/processes/" + electionId + "/ecdsa/auth"

  const body = {
    "authData": [signedElectionId]
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + orgApiToken,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body)
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { tokenR } = responseBody

  console.log("Get election shared key", tokenR)
  return tokenR
}

export async function getCspSigningTokenBlind(electionId: string, signedElectionId: string, orgApiToken: string): Promise<string> {
  const url = config.cspUrlPrefix + "/v1/auth/processes/" + electionId + "/blind/auth"

  const body = {
    "authData": [signedElectionId]
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + orgApiToken,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body)
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { tokenR } = responseBody

  console.log("Get election shared key", tokenR)
  return tokenR
}

export async function getCspPlainSignature(electionId: string, tokenR: string, payload: string, orgApiToken: string): Promise<string> {
  const url = config.cspUrlPrefix + "/v1/auth/processes/" + electionId + "/ecdsa/sign"

  const body = {
    tokenR,
    payload
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + orgApiToken,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body)
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { signature } = responseBody

  console.log("Get election shared key", signature)
  return signature
}

export async function getCspBlindSignature(electionId: string, tokenR: string, blindedPayload: string, orgApiToken: string): Promise<string> {
  const url = config.cspUrlPrefix + "/v1/auth/processes/" + electionId + "/blind/sign"

  const body = {
    tokenR,
    payload: blindedPayload
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + orgApiToken,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body)
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { signature } = responseBody

  console.log("Get election shared key", signature)
  return signature
}

//////////////////////////////////////////////////////////////////////////
// CSP endpoint calls - Custom
//////////////////////////////////////////////////////////////////////////

export async function getElectionSharedKeyCustom(electionId: string, proof: { param1: string, param2: string }, orgApiToken: string): Promise<string> {
  const url = config.cspUrlPrefix + "/v1/auth/processes/" + electionId + "/sharedKey"

  const body = {
    "authData": [proof.param1, proof.param2] // The custom values that the CSP expects
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + orgApiToken,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body)
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { sharedKey } = responseBody

  console.log("Get election shared key", sharedKey)
  return sharedKey
}

export async function getCspSigningTokenPlainCustom(electionId: string, proof: { param1: string, param2: string }, orgApiToken: string): Promise<string> {
  const url = config.cspUrlPrefix + "/v1/auth/processes/" + electionId + "/ecdsa/auth"

  const body = {
    "authData": [proof.param1, proof.param2] // The custom values that the CSP expects
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + orgApiToken,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body)
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { tokenR } = responseBody

  console.log("Get election shared key", tokenR)
  return tokenR
}

export async function getCspSigningTokenBlindCustom(electionId: string, proof: { param1: string, param2: string }, orgApiToken: string): Promise<string> {
  const url = config.cspUrlPrefix + "/v1/auth/processes/" + electionId + "/blind/auth"

  const body = {
    "authData": [proof.param1, proof.param2] // The custom values that the CSP expects
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Authorization": "Bearer " + orgApiToken,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body)
    // mode: 'cors', // no-cors, *cors, same-origin
    // credentials: 'same-origin', // include, *same-origin, omit
  })

  if (response.status != 200) {
    throw new Error(await response.text())
  }

  const responseBody = await response.json()

  const { error } = responseBody
  if (error) throw new Error(error)

  const { tokenR } = responseBody

  console.log("Get election shared key", tokenR)
  return tokenR
}
