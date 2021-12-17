import { createAnonymousElection, createOrganization, createSignedElection, deleteOrganization, getElectionPriv, getOrganizationPriv, listElectionsPriv, setOrganizationMetadata } from "./sections/integrator"
import { createIntegrator, deleteIntegrator } from "./sections/superadmin"
import { getElectionSecretInfoPub, getElectionListPub, getElectionInfoPub, getOrganizationPub, getElectionSharedKey, getElectionSharedKeyCustom, getCspSigningTokenPlain, getCspSigningTokenBlind, getCspSigningTokenPlainCustom, getCspSigningTokenBlindCustom, getCspPlainSignature, getCspBlindSignature, getBlindedPayload, getProofFromBlindSignature, getBallotPayload, submitBallot, getBallot } from "./sections/voter"
import { Wallet } from "@ethersproject/wallet"
import { wait } from "./util/wait"
import { ProcessKeys } from "@vocdoni/voting"

async function main() {
    // VOCDONI INTERNAL

    const { id: integratorId, apiKey: integratorApiKey } = await createIntegrator("Integrator Ltd")

    // INTEGRATOR ENDPOINTS (backend)

    const { organizationId, apiToken: orgApiToken } = await createOrganization("Association " + Math.random(), integratorApiKey)
    await wait(11)
    await getOrganizationPriv(organizationId, integratorApiKey)
    // resetOrganizationPublicKey()

    await setOrganizationMetadata(organizationId, integratorApiKey)
    await getOrganizationPriv(organizationId, integratorApiKey)
    await wait(11)

    const { electionId: electionId1 } = await createSignedElection(organizationId, integratorApiKey)
    const { electionId: electionId2 } = await createAnonymousElection(organizationId, integratorApiKey)
    await wait(11)
    // const electionList = await listElectionsPriv(organizationId, integratorApiKey)
    const election1Details = await getElectionPriv(electionId1, integratorApiKey)
    const election2Details = await getElectionPriv(electionId2, integratorApiKey)

    // VOTER ENDPOINTS (frontend)

    // const orgData = await getOrganizationPub(organizationId, orgApiToken)
    // const electionListPub = await getElectionListPub(electionId1, "active", orgApiToken)
    // const electionInfo1 = await getElectionInfoPub(electionId1, orgApiToken)

    // key for confidential election data
    // const cspSharedKey = await getElectionSharedKey(electionId1, signedElectionId, orgApiToken)
    const cspSharedKey = await getElectionSharedKeyCustom(electionId1, { param1: "123", param2: "234" }, orgApiToken)
    // const electionInfo2 = await getElectionSecretInfoPub(electionId2, cspSharedKey, orgApiToken)

    // NON ANONYMOUS AUTH
    // const tokenR = await getCspSigningTokenPlain(electionId1, signedElectionId, orgApiToken)
    // const tokenR = await getCspSigningTokenPlainCustom(electionId1, { param1: "123", param2: "234" }, orgApiToken)
    // const plainSignature = await getCspPlainSignature(electionId1, tokenR, payload, orgApiToken)

    // ANONYMOUS AUTH
    // const tokenR = await getCspSigningTokenBlind(electionId1, signedElectionId, orgApiToken)
    const tokenR = await getCspSigningTokenBlindCustom(electionId1, { param1: "123", param2: "234" }, orgApiToken)

    const wallet = Wallet.createRandom()
    const { hexBlinded: blindedPayload, userSecretData } = getBlindedPayload(electionId1, tokenR, wallet)

    const blindSignature = await getCspBlindSignature(electionId1, tokenR, blindedPayload, orgApiToken)
    const proof = getProofFromBlindSignature(blindSignature, userSecretData, wallet)

    const ballot = getBallotPayload(electionId1, proof, true, election1Details.ecryptionPubKeys)

    const { nullifier } = await submitBallot(electionId1, ballot, wallet, orgApiToken)
    const ballotDetails = await getBallot(nullifier, orgApiToken)

    // cleanup

    await deleteOrganization(organizationId, integratorId)
    await deleteIntegrator(integratorId)
}

main().catch(err => {
    console.error(err)
    process.exit(1)
})
