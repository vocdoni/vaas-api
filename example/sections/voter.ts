import "isomorphic-unfetch"
import { getConfig } from "../util/config"
import { hexStringToBuffer, strip0x } from "@vocdoni/common"
import { CAbundle, IProofCA, ProofCaSignatureTypes, SignedTx, Tx, VoteEnvelope } from "@vocdoni/data-models"
import { CensusBlind } from "@vocdoni/census"
import { ProcessKeys, Voting } from "@vocdoni/voting"
import { ProcessCensusOrigin } from "@vocdoni/contract-wrappers"
import { BytesSignature } from "@vocdoni/signing"
import { Wallet } from "@ethersproject/wallet"
import { hexlify } from "@ethersproject/bytes"
import { keccak256 } from "@ethersproject/keccak256"
import { UserSecretData } from "blindsecp256k1";

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
  const url = config.apiUrlPrefix + "/v1/pub/organizations/" + organizationId + "/elections/" + status

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
  const url = config.apiUrlPrefix + "/v1/pub/elections/" + electionId

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
  const url = config.apiUrlPrefix + "/v1/pub/elections/" + electionId + "/auth/" + cspSharedKey

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
// Vote and proof computation
//////////////////////////////////////////////////////////////////////////

export function getBlindedPayload(electionId: string, hexTokenR: string, ephemeralWallet: Wallet) {
  const tokenR = CensusBlind.decodePoint(hexTokenR)
  const caBundle = CAbundle.fromPartial({
    processId: new Uint8Array(hexStringToBuffer(electionId)),
    address: new Uint8Array(hexStringToBuffer(ephemeralWallet.address)),
  })

  // hash(bundle)
  const hexCaBundle = hexlify(CAbundle.encode(caBundle).finish())
  const hexCaHashedBundle = strip0x(keccak256(hexCaBundle))

  const { hexBlinded, userSecretData } = CensusBlind.blind(hexCaHashedBundle, tokenR)
  return { hexBlinded, userSecretData }
}

export function getProofFromBlindSignature(hexBlindSignature: string, userSecretData: UserSecretData, wallet: Wallet) {
  const unblindedSignature = CensusBlind.unblind(hexBlindSignature, userSecretData)

  const proof: IProofCA = {
    type: ProofCaSignatureTypes.ECDSA_BLIND,
    signature: unblindedSignature,
    voterAddress: wallet.address
  }

  return proof
}

export function getBallotPayload(processId: string, proof: IProofCA, hasEncryptedVotes: boolean, processKeys: ProcessKeys) {
  const choices = [1, 2, 3]

  if (hasEncryptedVotes) {
    return Voting.packageSignedEnvelope({
      censusOrigin: ProcessCensusOrigin.OFF_CHAIN_CA,
      votes: choices,
      censusProof: proof,
      processId,
      processKeys
    })
  }

  return Voting.packageSignedEnvelope({
    censusOrigin: ProcessCensusOrigin.OFF_CHAIN_CA,
    votes: choices,
    censusProof: proof,
    processId
  })
}

//////////////////////////////////////////////////////////////////////////
// CSP endpoint calls - Standard
//////////////////////////////////////////////////////////////////////////

export async function getElectionSharedKey(electionId: string, signedElectionId: string, orgApiToken: string): Promise<string> {
  const url = config.cspUrlPrefix + "/v1/auth/elections/" + electionId + "/sharedKey"

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
  const url = config.cspUrlPrefix + "/v1/auth/elections/" + electionId + "/ecdsa/auth"

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
  const url = config.cspUrlPrefix + "/v1/auth/elections/" + electionId + "/blind/auth"

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
  const url = config.cspUrlPrefix + "/v1/auth/elections/" + electionId + "/ecdsa/sign"

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
  const url = config.cspUrlPrefix + "/v1/auth/elections/" + electionId + "/blind/sign"

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
  const url = config.cspUrlPrefix + "/v1/auth/elections/" + electionId + "/sharedKey"

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
  const url = config.cspUrlPrefix + "/v1/auth/elections/" + electionId + "/ecdsa/auth"

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
  const url = config.cspUrlPrefix + "/v1/auth/elections/" + electionId + "/blind/auth"

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

//////////////////////////////////////////////////////////////////////////
// Vote delivery
//////////////////////////////////////////////////////////////////////////

export async function submitBallot(electionId: string, ballot: VoteEnvelope, ephemeralWallet: Wallet, orgApiToken: string) {
  // Prepare
  const tx = Tx.encode({ payload: { $case: "vote", vote: ballot } })
  const txBytes = tx.finish()

  const hexSignature = await BytesSignature.sign(txBytes, ephemeralWallet)
  const signature = new Uint8Array(Buffer.from(strip0x(hexSignature), "hex"))

  const signedTx = SignedTx.encode({ tx: txBytes, signature })
  const signedTxBytes = signedTx.finish()

  const base64Payload = Buffer.from(signedTxBytes).toString("base64")

  // Submit
  const url = config.apiUrlPrefix + "/v1/pub/elections/" + electionId + "/vote"

  const body = {
    "vote": [base64Payload]
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

  const { nullifier } = responseBody

  console.log("Submitted ballot", nullifier)
  return { nullifier }
}

export async function getBallot(nullifier: string, orgApiToken: string) {
  const url = config.apiUrlPrefix + "/v1/pub/nullifiers/" + nullifier

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

  const { electionId, registered, explorerUrl } = responseBody

  console.log("Get nullifier", nullifier, ":", responseBody)
  return { electionId, registered, explorerUrl }
}
