import { createAnonymousElection, createOrganization, createSignedElection, deleteOrganization, getElectionPriv, getOrganizationPriv, getOrganizationListPriv, listElectionsPriv, setOrganizationMetadata } from "./sections/integrator"
import { createIntegrator, deleteIntegrator } from "./sections/superadmin"
import { getElectionSecretInfoPub, getElectionListPub, getElectionInfoPub, getOrganizationPub, getElectionSharedKey, getElectionSharedKeyCustom, getCspSigningTokenPlain, getCspSigningTokenBlind, getCspSigningTokenPlainCustom, getCspSigningTokenBlindCustom, getCspPlainSignature,getProofFromPlainSignature, getCspBlindSignature, getBlindedPayload, getProofFromBlindSignature, getBallotPayload, submitBallot, getBallot, getPlainPayload } from "./sections/voter"
import { Wallet } from "@ethersproject/wallet"
import { wait } from "./util/wait"
import { fakeSign } from "./sections/fake-csp"

async function main() {
    const encryptedResults = true
    const confidential = true
    const anonymous = true

    // VOCDONI INTERNAL

    const { id: integratorId, apiKey: integratorApiKey } = await createIntegrator("Integrator Ltd")

    // INTEGRATOR ENDPOINTS (backend)

    const { organizationId, apiToken: orgApiToken } = await createOrganization("Association " + Math.random(), integratorApiKey)
    await getOrganizationPriv(organizationId, integratorApiKey)
    // resetOrganizationPublicKey()

    await setOrganizationMetadata(organizationId, integratorApiKey)
    await getOrganizationPriv(organizationId, integratorApiKey)
    await getOrganizationListPriv(integratorApiKey)

    var electionId: string
    if (anonymous) {
       electionId = await createAnonymousElection(organizationId, encryptedResults, confidential, integratorApiKey)
    } else {
        electionId  = await createSignedElection(organizationId, encryptedResults, confidential, integratorApiKey)
    }

    const electionList = await listElectionsPriv(organizationId, integratorApiKey)
    const electionDetailsPriv = await getElectionPriv(electionId, integratorApiKey)

    // VOTER ENDPOINTS (frontend)

    // const orgData = await getOrganizationPub(organizationId, orgApiToken)
    // const electionListPub = await getElectionListPub(electionId, "active", orgApiToken)
    // const electionInfo = await getElectionInfoPub(electionId, orgApiToken)

    const wallet = Wallet.createRandom()
    const voterId = "000000000000000000000000" + wallet.address.slice(2)
    const signature = fakeSign(electionId, voterId)

    // key for confidential election data
    // const cspSharedKey = await getElectionSharedKey(electionId1, signedElectionId, orgApiToken)
    const cspSharedKey = await getElectionSharedKeyCustom(electionId, { voterId, signature }, orgApiToken)
    
    let election1DetailsPubAuth = await getElectionSecretInfoPub(electionId, cspSharedKey, orgApiToken)
    while (election1DetailsPubAuth.status == "UPCOMING") {
        await wait(5)
        election1DetailsPubAuth = await getElectionSecretInfoPub(electionId, cspSharedKey, orgApiToken)
    }

    var proof
    if (anonymous) {
        // ANONYMOUS AUTH
        const tokenR = await getCspSigningTokenBlindCustom(electionId, { voterId, signature }, orgApiToken)
    
        const { hexBlinded: blindedPayload, userSecretData } = getBlindedPayload(electionId, tokenR, wallet)
    
        const blindSignature = await getCspBlindSignature(electionId, tokenR, blindedPayload, orgApiToken)
        proof = getProofFromBlindSignature(blindSignature, userSecretData, wallet)
    } else {
        // NON ANONYMOUS AUTH
        const tokenR = await getCspSigningTokenPlain(electionId, signature, orgApiToken)
        const payload = await getPlainPayload(electionId, tokenR, wallet)
        const plainSignature = await getCspPlainSignature(electionId, tokenR, payload, orgApiToken)
        proof = getProofFromPlainSignature(plainSignature, wallet)
    }




    const ballot = getBallotPayload(electionId, proof, encryptedResults, election1DetailsPubAuth.encryptionPubKeys)
    const { nullifier } = await submitBallot(electionId, election1DetailsPubAuth.chainId, ballot, wallet, orgApiToken)
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
