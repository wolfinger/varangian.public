# varangian - aws + shopify of investment managment

## overview

todo: write this section

## core services

- accounting
- performance
- portfolio management
- risk
- tax

## resources

- organizations (orgs)
- users
- accounts (accts)
- portfolios (ports)
- strategies (strats)
- instruments (insts)
- transactions (txns)
- lots

TODO: update services to support processing arrays of data (e.g., insert multiple instruments in one request)

###  organizations

an `org` can be any type of entity. orgs are self-referential to create org hierarchies.

tablename: `orgs`

| field       | type      | key        | not null | description                   |
| ----------- | --------- | ---------- | -------- | ----------------------------- |
| id          | `vxid`    | pk         | x        | unique vxid for each org record. org ids begin with the `org` prefix. |
| name        | `text`    |            |          | alphanumeric name for the org. |
| parent_id   | `vxid`    | fk(`orgs`) |          | vxid linking the org to a parent. null if this is the parent org. |

### users

a `user` is an varangian account.

tablename: `users`

| field       | type      | key        | not null | description                   |
| ----------- | --------- | ---------- | -------- | ----------------------------- |
| id          | `vxid`    | pk         | x        | unique vxid for each user record. user ids begin with the `usr` prefix. |
| login_name  | `text`    |            |          | alphanumeric user name to allow for human-friendly log-ins. |

### accounts

an `account` is any representation of a group of lots, typically to represent positions held with a specific custody bank or brokerage.

tablename: `accts`

| field       | type      | key        | not null | description                   |
| ----------- | --------- | ---------- | -------- | ----------------------------- |
| id          | `vxid`    | pk         | x        | unique vxid for each account record. account ids begin with the `acct` prefix. |
| name        | `text`    |            |          | alphanumeric name for the account. |
| parent_id   | `vxid`    | fk(`accts`) |         | vxid linking the account to a parent. null if this is the parent account. useful if a broker/custody bank has subaccounts and stuff. | 

### portfolios

a `portfolio` is a logical group of lots. portfolios are meant to group lots logically regarless of how they are held in reality.

tablename: `ports`

| field       | type      | key        | not null | description                   |
| ----------- | --------- | ---------- | -------- | ----------------------------- |
| id          | `vxid`    | pk         | x        | unique vxid for each portfolio. portfolio ids begin with the `prt` prefix. |
| name        | `text`    |            |          | alphanumeric name for the portfolio. |
| parent_id   | `vxid`    | fk(`ports`) |         | vxid linking the portfolio to a parent. null if this is the top level. |

### strategies

a `strategy` is a logical group of lots associated with a specific strategy. strategies are meant to group lots to track specific strategies employed in a portfolio or across multiple portfolios.

tablename: `strats`

| field       | type      | key        | not null | description                   |
| ----------- | --------- | ---------- | -------- | ----------------------------- |
| id          | `vxid`    | pk         | x        | unique vxid for each strategy. strategy ids begin with the `str` prefix. |
| name        | `text`    |            |          | alphanumeric name for the strategy. |
| parent_id   | `vxid`    | fk(`strats`) |        | vxid linking the strategy to a parent. null if this is the top level. |

### instruments

an `instrument` is any object that can be priced and/or traded.

tablename: `insts`

| field       | type      | key        | not null | description                   |
| ----------- | --------- | ---------- | -------- | ----------------------------- |
| id          | `vxid`    | pk         | x        | unique vxid for each instrument record. instrument ids begin with the `inst` prefix. |
| ticker_local  | `text`  |            |          | market-accepted ticker in the local jurisdiction. |
| ticker_vgn    | `text`  |            |          | varangian ticker ... todo: make dynamic based on tags | 
| proxy_inst  | `vxid`    | fk(`insts`) |     | vxid linking to a proxy instrument. used for instruments that don't have full instrument support. |

todo: determine how to setup look-thru instruments (e.g., underlying fund holdings)

### transactions

a `transaction` is a record of an activity that changes state for something in varangian.

tablename: `txns`

| field       | type      | key        | not null | description                   |
| ----------- | --------- | ---------- | -------- | ----------------------------- |
| id          | `vxid`    | pk         | x        | unique vxid for each transaction record. transaction ids begin with the `txn` prefix. |
| txn_dt      | `timestamptz` |        |          | date and time of the transaction |
| settle_dt   | `timestamptz` |        |          | date for the transaction to settle |
| txn_type    | `text`    |            |          | type of the transaction being processed (see below) |
| txn_sub_type | `text`   |            |          | sub type of the transaction being processed (see below)
| txn_size    | `float8`  |            |          | notional size of the transaction |
| inst_id     | `vxid`    | fk(`insts`) |         | vxid of the instrument involved in the transaction |
| parent_id   | `vxid`    | fk(`txns`) |          | vxid linking to a parent txn. null if there is no parent |
| lot_id      | `vxid`    | fk(`lots`) |          | vxid linking txn to a specific lot (e.g., allocating transactions) |
| state       | `text`    |            |          | state of the transaction. used primarily to indicate unprocessed txns |

| trade_amt_ccy | `vxid` | fk(`insts`) |          | vxid of the trade currency
| trade_amt_gross | `float8` |         |          | gross trade amount
| trade_amt_net | `float8` |           |          | net (of fees) trade amount
| settle_amt_ccy | `vxid` | fk(`insts`) |         | vxid of the settlement currency
| settle_amt_gross | `float8` |        |          | gross settle amount
| settle_amt_net | `float8` |          |          | net (of fees) settle amount

`txn_type`
- `multileg` - parent transaction of a package of transactions
- `trade` - buy, sell, buy (reinvest)
- `settle` - settlement for a trade
- `sweep` - movement of cash into or out of a sweep vehicle (e.g., mmf)
- `xfer` - transfer in to or out of an account (xfin, xfout)
- `corpact` - corporate action (e.g., stock split, dividend)
  
TODO: maybe create sub accounts for each account that are liability and asset accounts so it fits the accounting identities  
TODO: activities are cash / operations basis or accrual basis ... is that a transaction type, a new 'type`, or account based?

#### transaction process flows

- trade
    - buy
    - sell
    - reinvest
- settle
- income
    - dividend
    - interest
- sweep
    - in
    - out
- transfer
- allocation

##### `trade`

buy:  
payable for txn amount to lots  
unsettled share amount to lots  
// pending settlement based on settle date to pending activity ledger  
sweep out funds to cash, send cash, release payable  
receive shares, update settle amount to lots  
  
sell:  
receivable for txn amount to lots  
unsettled share amount to lots  
// pending settlement based on setttle date to pending activity ledger  
receive cash, sweep in to sweep vehicle, release receivable  
send shares, update settle amount to lots  

##### `allocation`

an allocating transaction allocates a parent transaction to specific lots (e.g., a sale that is applied to multiple lots). each allocating transaction has a `parent_id` referring to the parent transaction and a `tgt_lot_id` specifying the target allocation  
  
allocation transactions are typically generated automatically to create an audit trail when processing another transaction

##### `settle`

id - unique  
txnDt - settle date  
settleDt - settle date  
txnType - settle  
txnSubType - [null]  
parent_id - txn being settled (logic to find this)  
inst_id - from parent txn  
  
pull allocating transactions to determine which lots to settle against  
update each allocated lot with new settled amount  
update payable/receivable balance to 0  
update cash balance to net new balance

##### `dividend`

latest thinking: attach income receivables / payables to the lot itself  
  
dividend receivable generated for each lot on ex date (for lots held on record date)  
receive cash, sweep in to sweep vehicle, release receivable  
  
optional dividend reinvesment:  
buy fractional shares  

##### `transfer`

transfer a lot of something into or out of an account  
// how do i handle transfers without defining a single account?

### lots

a `lot` is the atomic unit in varangian.  
  
tablename: `lots`  
  
core:
| field       | type      | key        | not null | description                   |
| ----------- | --------- | ---------- | -------- | ----------------------------- |
| id          | `vxid`    | pk         | x        | unique vxid for each lot record. lot ids begin with the `lot` prefix. |
| inst_id     | `vxid`    | fk(`insts`) |         | foreign key to the unique instrument vxid. instruments begin with the `inst` prefix. |
| src_txn_id  | `vxid`    | fk(`txns`) |          | vxid linking the lot to the transaction record. begin with the `txn` prefix. |
| orig_dt     | `timestamptz`|         |          | the original txn timestamp. useful for txn tracing, tax lot optimizing, etc. |
| orig_size   | `float8`  |            |          | the original txn lot size. see point-in-time section for tracking size over time. |
| le_org_id   | `vxid`    | fk(`orgs`) |          | foreign key to the legal entity org that owns the lot. orgs begin with the `org` prefix. |
| acct_id     | `vxid`    | fk(`accts`) |         | foreign key to the account where the lot is held. accounts begin with the `acct` prefix. |

lot balances at a point-in-time (`lot_bals`):
| field       | type      | key        | not null | description                   |
| ----------- | --------- | ---------- | -------- | ----------------------------- |
| lot_id      | `vxid`    | pk, fk(`lots`) | x    | foreign key to the lots table. |
| lot_dt      | `timestamptz` | pk     | x        | timestamp for the lot at a point in time. transaciton logs can recreate lot properties over time. |
| lot_size    | `float8`  |            |          | lot size. depending on the instrument type this is equivalent to shares, notional, etc. lot size is the net of the settled and unsettled size. |
| settled_size | `float8` |            |          | size of lot that's been settled. |
| unsettled_size | `float8` |          |          | size of the lot that hasn't been settled yet. |

TODO: determine if lot balances should be designed as a singleton w/ access as `/lots/{id}/balance`


## other functionality

### key generation

varagian takes a hybrid approach to key management. internally all keys are left to the data store to generate. this allows any service to operate with the data store indpendently and not have to worry about key generation.

varangian uses the native uuid type in postgres to generate v4 uuids. these ids are used when interacting with the internal system, but are converted to external ids when used with public facing services.

the `vxid` package supports working between public and private varangian keys. since uuid v4 keys are well known (or can be easily searched for on the interent), i will not spend time discussing them. instead i will focus on the public keys which are meant to be consumed by all varangian services whether exposed to the public or not.

an `vxid` is a base57 encoded URI safe uuid with a prefix attached to assist with human readable debugging, logging, and tracing. prefixes typically use an abbreviation with an associated resource.

| prefix | object type    |
| ------ | -------------- |
| org    | organization   |
| usr    | user           |
| acct   | account        |
| prt    | portfolio      |
| str    | strategy       |
| inst   | instrument     |
| txn    | transaction    |
| lot    | lot            |

### oinst (open instruments)

the oinst, or open instruments, service is a crowd-sourced database for financial instruments. data providers are becoming more and more extractionary in the value chain and oinst is a solution to combat this behavior. varangian's philosophy is to focus on value creation for the ecosystem instead of value extraction unlike most players in the financial services industry.

varangian creates a network effect for financial data by enabling a platform for investors to contribute to open financial data. the higher the quality of data on oinst, the greater the use by investors and therefore the greater contributions, which create a virtous flywheel effect.

at some point it may be worth looking at a crytpto economic model for this service to reward those who contribute more value to the service.