import { createSnapshot, restoreSnapshot } from "./helpers/snapshot";

const Registry = artifacts.require('./Registry.sol')
const TokenStaking = artifacts.require('./TokenStakingStub.sol')
const KeepBonding = artifacts.require('./KeepBonding.sol')
const TestEtherReceiver = artifacts.require('./TestEtherReceiver.sol')

const { expectRevert } = require('openzeppelin-test-helpers');

const BN = web3.utils.BN

const chai = require('chai')
chai.use(require('bn-chai')(BN))
const expect = chai.expect

contract('KeepBonding', (accounts) => {
    let registry
    let tokenStaking   
    let keepBonding
    let etherReceiver

    let operator
    let authorizer
    let bondCreator
    let sortitionPool

    before(async () => {
        operator = accounts[1]
        authorizer = operator
        bondCreator = accounts[4]   
        sortitionPool = accounts[5]

        registry = await Registry.new()
        tokenStaking = await TokenStaking.new()
        keepBonding = await KeepBonding.new(registry.address, tokenStaking.address)
        etherReceiver = await TestEtherReceiver.new()

        await registry.approveOperatorContract(bondCreator)
        await keepBonding.authorizeSortitionPoolContract(operator, sortitionPool, {from: authorizer})

        await tokenStaking.authorizeOperatorContract(operator, bondCreator)
    })

    beforeEach(async () => {
        await createSnapshot()
    })

    afterEach(async () => {
        await restoreSnapshot()
    })

    describe('deposit', async () => {
        it('registers unbonded value', async () => {
            const value = new BN(100)

            const expectedUnbonded = value

            await keepBonding.deposit(operator, { value: value })

            const unbonded = await keepBonding.availableUnbondedValue(operator, bondCreator, sortitionPool)

            expect(unbonded).to.eq.BN(expectedUnbonded, 'invalid unbonded value')
        })
    })

    describe('withdraw', async () => {
        const destination = accounts[2]
        const value = new BN(1000)

        beforeEach(async () => {
            await keepBonding.deposit(operator, { value: value })
        })

        it('transfers unbonded value', async () => {
            const expectedUnbonded = 0
            const expectedDestinationBalance = web3.utils.toBN(await web3.eth.getBalance(destination)).add(value)

            await keepBonding.withdraw(value, destination, { from: operator })

            const unbonded = await keepBonding.availableUnbondedValue(operator, bondCreator, sortitionPool)
            expect(unbonded).to.eq.BN(expectedUnbonded, 'invalid unbonded value')

            const destinationBalance = await web3.eth.getBalance(destination)
            expect(destinationBalance).to.eq.BN(expectedDestinationBalance, 'invalid destination balance')
        })

        it('fails if insufficient unbonded value', async () => {
            const invalidValue = value.add(new BN(1))

            await expectRevert(
                keepBonding.withdraw(invalidValue, destination, { from: operator }),
                "Insufficient unbonded value"
            )
        })

        it('reverts if transfer fails', async () => {
            await etherReceiver.setShouldFail(true)

            await expectRevert(
                keepBonding.withdraw(value, etherReceiver.address, { from: operator }),
                "Transfer failed"
            )
        })
    })

    describe('availableUnbondedValue', async () => {
        const value = new BN(100)

        beforeEach(async () => {
            await keepBonding.deposit(operator, { value: value })            
        })

        it('returns zero for operator with no deposit', async () => {
            const unbondedOperator = "0x0000000000000000000000000000000000000001"
            const expectedUnbonded = 0

            const unbondedValue = await keepBonding.availableUnbondedValue(unbondedOperator, bondCreator, sortitionPool)
            expect(unbondedValue).to.eq.BN(expectedUnbonded, 'invalid unbonded value')
        })

        it('return zero when bond creator is not approved by operator', async () => {
            const notApprovedBondCreator = "0x0000000000000000000000000000000000000001"
            const expectedUnbonded = 0

            const unbondedValue = await keepBonding.availableUnbondedValue(operator, notApprovedBondCreator, sortitionPool)
            expect(unbondedValue).to.eq.BN(expectedUnbonded, 'invalid unbonded value')
        })

        it('returns zero when sortition pool is not authorized', async () => {
            const notAuthorizedSortitionPool = "0x0000000000000000000000000000000000000001"
            const expectedUnbonded = 0

            const unbondedValue = await keepBonding.availableUnbondedValue(operator, bondCreator, notAuthorizedSortitionPool)
            expect(unbondedValue).to.eq.BN(expectedUnbonded, 'invalid unbonded value') 
        })

        it('returns value of operators deposit', async () => {
            const expectedUnbonded = value

            const unbonded = await keepBonding.availableUnbondedValue(operator, bondCreator, sortitionPool)

            expect(unbonded).to.eq.BN(expectedUnbonded, 'invalid unbonded value')
        })
    })

    describe('createBond', async () => {
        const holder = accounts[3]
        const value = new BN(100)

        beforeEach(async () => {
            await keepBonding.deposit(operator, { value: value })
        })

        it('creates bond', async () => {
            const reference = 888

            const expectedUnbonded = 0

            await keepBonding.createBond(operator, holder, reference, value, sortitionPool, {from: bondCreator})

            const unbonded = await keepBonding.availableUnbondedValue(operator, bondCreator, sortitionPool)
            expect(unbonded).to.eq.BN(expectedUnbonded, 'invalid unbonded value')

            const lockedBonds = await keepBonding.bondAmount(operator, holder, reference)
            expect(lockedBonds).to.eq.BN(value, 'unexpected bond value')
        })

        it('creates two bonds with the same reference for different operators', async () => {
            const operator2 = accounts[2]
            const authorizer2 = accounts[2]
            const bondValue = new BN(10)
            const reference = 777

            const expectedUnbonded = value.sub(bondValue)

            await keepBonding.deposit(operator2, { value: value })

            await tokenStaking.authorizeOperatorContract(operator2, bondCreator)
            await keepBonding.authorizeSortitionPoolContract(operator2, sortitionPool, {from: authorizer2})
            await keepBonding.createBond(operator, holder, reference, bondValue, sortitionPool, {from: bondCreator})
            await keepBonding.createBond(operator2, holder, reference, bondValue, sortitionPool, {from: bondCreator})

            const unbonded1 = await keepBonding.availableUnbondedValue(operator, bondCreator, sortitionPool)
            expect(unbonded1).to.eq.BN(expectedUnbonded, 'invalid unbonded value 1')

            const unbonded2 = await keepBonding.availableUnbondedValue(operator2, bondCreator, sortitionPool)
            expect(unbonded2).to.eq.BN(expectedUnbonded, 'invalid unbonded value 2')

            const lockedBonds1 = await keepBonding.bondAmount(operator, holder, reference)
            expect(lockedBonds1).to.eq.BN(bondValue, 'unexpected bond value 1')

            const lockedBonds2 = await keepBonding.bondAmount(operator2, holder, reference)
            expect(lockedBonds2).to.eq.BN(bondValue, 'unexpected bond value 2')
        })

        it('fails to create two bonds with the same reference for the same operator', async () => {
            const bondValue = new BN(10)
            const reference = 777

            await keepBonding.createBond(operator, holder, reference, bondValue, sortitionPool, {from: bondCreator})

            await expectRevert(
                keepBonding.createBond(operator, holder, reference, bondValue, sortitionPool, {from: bondCreator}),
                "Reference ID not unique for holder and operator"
            )
        })

        it('fails if insufficient unbonded value', async () => {
            const bondValue = value.add(new BN(1))

            await expectRevert(
                keepBonding.createBond(operator, holder, 0, bondValue, sortitionPool, {from: bondCreator}),
                "Insufficient unbonded value"
            )
        })
    })

    describe('reassignBond', async () => {
        const holder = accounts[2]
        const newHolder = accounts[3]
        const bondValue = new BN(100)
        const reference = 777
        const newReference = 888

        beforeEach(async () => {
            await keepBonding.deposit(operator, { value: bondValue })
            await keepBonding.createBond(operator, holder, reference, bondValue, sortitionPool, {from: bondCreator})
        })

        it('reassigns bond to a new holder and a new reference', async () => {
            await keepBonding.reassignBond(operator, reference, newHolder, newReference, { from: holder })

            let lockedBonds = await keepBonding.bondAmount(operator, holder, reference)
            expect(lockedBonds).to.eq.BN(0, 'invalid locked bonds')

            lockedBonds = await keepBonding.bondAmount(operator, holder, newReference)
            expect(lockedBonds).to.eq.BN(0, 'invalid locked bonds')

            lockedBonds = await keepBonding.bondAmount(operator, newHolder, reference)
            expect(lockedBonds).to.eq.BN(0, 'invalid locked bonds')

            lockedBonds = await keepBonding.bondAmount(operator, newHolder, newReference)
            expect(lockedBonds).to.eq.BN(bondValue, 'invalid locked bonds')
        })

        it('reassigns bond to the same holder and a new reference', async () => {
            await keepBonding.reassignBond(operator, reference, holder, newReference, { from: holder })

            let lockedBonds = await keepBonding.bondAmount(operator, holder, reference)
            expect(lockedBonds).to.eq.BN(0, 'invalid locked bonds')

            lockedBonds = await keepBonding.bondAmount(operator, holder, newReference)
            expect(lockedBonds).to.eq.BN(bondValue, 'invalid locked bonds')
        })

        it('reassigns bond to a new holder and the same reference', async () => {
            await keepBonding.reassignBond(operator, reference, newHolder, reference, { from: holder })

            let lockedBonds = await keepBonding.bondAmount(operator, holder, reference)
            expect(lockedBonds).to.eq.BN(0, 'invalid locked bonds')

            lockedBonds = await keepBonding.bondAmount(operator, newHolder, reference)
            expect(lockedBonds).to.eq.BN(bondValue, 'invalid locked bonds')
        })

        it('fails if sender is not the holder', async () => {
            await expectRevert(
                keepBonding.reassignBond(operator, reference, newHolder, newReference, { from: accounts[0] }),
                "Bond not found"
            )
        })

        it('fails if reassigned to the same holder and the same reference', async () => {
            await keepBonding.deposit(operator, { value: bondValue })
            await keepBonding.createBond(operator, holder, newReference, bondValue, sortitionPool, {from: bondCreator})

            await expectRevert(
                keepBonding.reassignBond(operator, reference, holder, newReference, { from: holder }),
                "Reference ID not unique for holder and operator"
            )
        })
    })

    describe('freeBond', async () => {
        const holder = accounts[2]
        const bondValue = new BN(100)
        const reference = 777

        beforeEach(async () => {
            await keepBonding.deposit(operator, { value: bondValue })
            await keepBonding.createBond(operator, holder, reference, bondValue, sortitionPool, {from: bondCreator})
        })

        it('releases bond amount to operator\'s available bonding value', async () => {
            await keepBonding.freeBond(operator, reference, { from: holder })

            const lockedBonds = await keepBonding.bondAmount(operator, holder, reference)
            expect(lockedBonds).to.eq.BN(0, 'unexpected remaining locked bonds')

            const unbondedValue = await keepBonding.availableUnbondedValue(operator, bondCreator, sortitionPool)
            expect(unbondedValue).to.eq.BN(bondValue, 'unexpected unbonded value')
        })

        it('fails if sender is not the holder', async () => {
            await expectRevert(
                keepBonding.freeBond(operator, reference, { from: accounts[0] }),
                "Bond not found"
            )
        })
    })

    describe('seizeBond', async () => {
        const holder = accounts[2]
        const destination = accounts[3]
        const bondValue = new BN(1000)
        const reference = 777

        beforeEach(async () => {
            await keepBonding.deposit(operator, { value: bondValue })
            await keepBonding.createBond(operator, holder, reference, bondValue, sortitionPool, {from: bondCreator})
        })

        it('transfers whole bond amount to destination account', async () => {
            const amount = bondValue
            let expectedBalance = web3.utils.toBN(await web3.eth.getBalance(destination)).add(amount)

            await keepBonding.seizeBond(operator, reference, amount, destination, { from: holder })

            const actualBalance = await web3.eth.getBalance(destination)
            expect(actualBalance).to.eq.BN(expectedBalance, 'invalid destination account balance')

            const lockedBonds = await keepBonding.bondAmount(operator, holder, reference)
            expect(lockedBonds).to.eq.BN(0, 'unexpected remaining bond value')
        })

        it('transfers less than bond amount to destination account', async () => {
            const remainingBond = new BN(1)
            const amount = bondValue.sub(remainingBond)
            let expectedBalance = web3.utils.toBN(await web3.eth.getBalance(destination)).add(amount)

            await keepBonding.seizeBond(operator, reference, amount, destination, { from: holder })

            const actualBalance = await web3.eth.getBalance(destination)
            expect(actualBalance).to.eq.BN(expectedBalance, 'invalid destination account balance')

            const lockedBonds = await keepBonding.bondAmount(operator, holder, reference)
            expect(lockedBonds).to.eq.BN(remainingBond, 'unexpected remaining bond value')
        })

        it('reverts if seized amount equals zero', async () => {
            const amount = new BN(0)
            await expectRevert(
                keepBonding.seizeBond(operator, reference, amount, destination, { from: holder }),
                "Requested amount should be greater than zero"
            )
        })

        it('reverts if seized amount is greater than bond value', async () => {
            const amount = bondValue.add(new BN(1))
            await expectRevert(
                keepBonding.seizeBond(operator, reference, amount, destination, { from: holder }),
                "Requested amount is greater than the bond"
            )
        })

        it('reverts if transfer fails', async () => {
            await etherReceiver.setShouldFail(true)
            const destination = etherReceiver.address

            await expectRevert(
                keepBonding.seizeBond(operator, reference, bondValue, destination, { from: holder }),
                "Transfer failed"
            )

            const destinationBalance = await web3.eth.getBalance(destination)
            expect(destinationBalance).to.eq.BN(0, 'invalid destination account balance')

            const lockedBonds = await keepBonding.bondAmount(operator, holder, reference)
            expect(lockedBonds).to.eq.BN(bondValue, 'unexpected bond value')
        })
    })

    describe("authorizeSortitionPoolContract", async () => {
        it("reverts when operator is not an authorizer", async () => {
            let authorizer1 = accounts[2]

            await expectRevert(
                keepBonding.authorizeSortitionPoolContract(operator, sortitionPool, { from: authorizer1 }),
                'Not authorized'
            )
        })

        it("should authorize sortition pool for provided operator", async () => {
            await keepBonding.authorizeSortitionPoolContract(operator, sortitionPool, { from: authorizer })
            
            assert.isTrue(
                await keepBonding.hasSecondaryAuthorization(operator, sortitionPool), 
                "Sortition pool has not beeen authorized for provided operator"
            )
        })
    })
})
