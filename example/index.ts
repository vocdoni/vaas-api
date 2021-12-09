import { createAnonymousElection, createOrganization, createSignedElection, deleteOrganization, getElectionPriv, getOrganizationPriv, listElectionsPriv, setOrganizationMetadata } from "./sections/integrator"
import { createIntegrator, deleteIntegrator } from "./sections/superadmin"

async function main() {
    // VOCDONI INTERNAL

    const { id: integratorId, apiKey: integratorApiKey } = await createIntegrator("Integrator Ltd")

    // INTEGRATOR ENDPOINTS (backend)

    const { organizationId, apiToken: orgApiToken } = await createOrganization("Association " + Math.random(), integratorApiKey)
    await getOrganizationPriv(organizationId, integratorApiKey)
    // resetOrganizationPublicKey()

    await setOrganizationMetadata(organizationId, integratorApiKey)
    await getOrganizationPriv(organizationId, integratorApiKey)

    const { electionId: electionId1 } = await createSignedElection(organizationId, integratorApiKey)
    const { electionId: electionId2 } = await createAnonymousElection(organizationId, integratorApiKey)
    // const electionList = await listElectionsPriv(organizationId, integratorApiKey)
    // const electionDetails = await getElectionPriv(electionId1, integratorApiKey)

    // VOTER ENDPOINTS (frontend)

    // getOrganization()
    // getElectionList()
    // getElectionInfoPublic()

    // getElectionSharedKey()    // for confidential elections
    // getElectionInfoConfidential(sharedKey)

    // getCspBlindingToken()
    // getCspBlindSignature()

    // submitBallot()
    // getBallot()


    // cleanup

    await deleteOrganization(organizationId, orgApiToken)
    await deleteIntegrator(integratorId)
}

main().catch(err => {
    console.error(err)
    process.exit(1)
})
