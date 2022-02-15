import { createAnonymousElection, createOrganization, createSignedElection, deleteOrganization, getElectionPriv, getOrganizationPriv, getOrganizationListPriv, listElectionsPriv, setOrganizationMetadata } from "./sections/integrator"
import { createIntegrator, deleteIntegrator } from "./sections/superadmin"
import { getElectionSecretInfoPub, getElectionListPub, getElectionInfoPub, getOrganizationPub, getElectionSharedKey, getElectionSharedKeyCustom, getCspSigningTokenPlain, getCspSigningTokenBlind, getCspSigningTokenPlainCustom, getCspSigningTokenBlindCustom, getCspPlainSignature, getCspBlindSignature, getBlindedPayload, getProofFromBlindSignature, getBallotPayload, submitBallot, getBallot } from "./sections/voter"
import { Wallet } from "@ethersproject/wallet"
import { wait } from "./util/wait"
import { fakeSign } from "./sections/fake-csp"

async function main() {
    const encryptedResults = true
    const confidential = true

    // VOCDONI INTERNAL

    const { id: integratorId, apiKey: integratorApiKey } = await createIntegrator("Integrator Ltd")

    // INTEGRATOR ENDPOINTS (backend)

    const { organizationId, apiToken: orgApiToken } = await createOrganization("Association " + Math.random(), integratorApiKey)
    await getOrganizationPriv(organizationId, integratorApiKey)
    // resetOrganizationPublicKey()

    await setOrganizationMetadata(organizationId, integratorApiKey)
    await getOrganizationPriv(organizationId, integratorApiKey)
    await getOrganizationListPriv(integratorApiKey)

    const { electionId: electionId1 } = await createSignedElection(organizationId, encryptedResults, confidential, integratorApiKey)
    // const { electionId: electionId2 } = await createAnonymousElection(organizationId, encryptedResults, confidential, integratorApiKey)

    const electionList = await listElectionsPriv(organizationId, integratorApiKey)
    const election1DetailsPriv = await getElectionPriv(electionId1, integratorApiKey)
    // const election2Details = await getElectionPriv(electionId2, integratorApiKey)

    // VOTER ENDPOINTS (frontend)

    // const orgData = await getOrganizationPub(organizationId, orgApiToken)
    // const electionListPub = await getElectionListPub(electionId1, "active", orgApiToken)
    // const electionInfo1 = await getElectionInfoPub(electionId1, orgApiToken)

    const wallet = Wallet.createRandom()
    const voterId = "000000000000000000000000" + wallet.address.slice(2)
    const signature = fakeSign(electionId1, voterId)

    // key for confidential election data
    // const cspSharedKey = await getElectionSharedKey(electionId1, signedElectionId, orgApiToken)
    const cspSharedKey = await getElectionSharedKeyCustom(electionId1, { voterId, signature }, orgApiToken)
    // const electionInfo2 = await getElectionSecretInfoPub(electionId2, cspSharedKey, orgApiToken)
    
    let election1DetailsPubAuth = await getElectionSecretInfoPub(electionId1, cspSharedKey, orgApiToken)
    while (election1DetailsPubAuth.status == "UPCOMING") {
        await wait(5)
        election1DetailsPubAuth = await getElectionSecretInfoPub(electionId1, cspSharedKey, orgApiToken)
    }

    // NON ANONYMOUS AUTH
    // const tokenR = await getCspSigningTokenPlain(electionId1, signedElectionId, orgApiToken)
    // const tokenR = await getCspSigningTokenPlainCustom(electionId1, { voterId, signature }, orgApiToken)
    // const plainSignature = await getCspPlainSignature(electionId1, tokenR, payload, orgApiToken)

    // ANONYMOUS AUTH
    // const tokenR = await getCspSigningTokenBlind(electionId1, signedElectionId, orgApiToken)
    const tokenR = await getCspSigningTokenBlindCustom(electionId1, { voterId, signature }, orgApiToken)

    const { hexBlinded: blindedPayload, userSecretData } = getBlindedPayload(electionId1, tokenR, wallet)

    const blindSignature = await getCspBlindSignature(electionId1, tokenR, blindedPayload, orgApiToken)
    const proof = getProofFromBlindSignature(blindSignature, userSecretData, wallet)
    const ballot = getBallotPayload(electionId1, proof, encryptedResults, election1DetailsPubAuth.encryptionPubKeys)
    const { nullifier } = await submitBallot(electionId1, ballot, wallet, orgApiToken)
    let ballotDetails = await getBallot(nullifier, orgApiToken)
    // optionally wait for the ballot to be registered if not already
    // while (!ballotDetails.registered) {
    //     await wait(5)
    //     ballotDetails =await getBallot(nullifier, orgApiToken)
    // }

    // cleanup

    await deleteOrganization(organizationId, integratorApiKey)
    await deleteIntegrator(integratorId)
}

main().catch(err => {
    console.error(err)
    process.exit(1)
})
