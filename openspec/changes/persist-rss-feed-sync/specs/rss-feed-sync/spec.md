## ADDED Requirements

### Requirement: Service synchronizes configured RSS feed sources
The system SHALL load configured RSS/Atom feed sources, fetch enabled sources on manual or scheduled runs, and process each source independently so one source failure does not block the others.

#### Scenario: Scheduled sync processes enabled sources
- **WHEN** RSS sync is enabled and the scheduler interval elapses
- **THEN** the service fetches each enabled feed source once in that run and records a result for every source

#### Scenario: Single source manual sync
- **WHEN** an operator triggers sync for a specific feed source
- **THEN** the service syncs only that source and leaves other configured sources untouched

### Requirement: Service persists feed content idempotently
The system SHALL persist feed source metadata and feed entry content using deterministic uniqueness rules so repeated syncs do not create duplicate entries for the same source item.

#### Scenario: Re-fetching existing entries
- **WHEN** a feed returns items that were already persisted for the same source
- **THEN** the service updates the existing records in place instead of inserting duplicates

#### Scenario: Feed item lacks canonical guid
- **WHEN** a feed item does not provide a GUID or canonical entry ID
- **THEN** the service derives a stable fallback identity from the configured key order and still performs idempotent persistence

### Requirement: Service tracks per-source sync state
The system SHALL persist sync checkpoint and run status for each feed source, including last success time, last run status, latest error, and HTTP cache validators when available.

#### Scenario: Successful source sync updates checkpoint
- **WHEN** a feed source sync completes successfully
- **THEN** the system stores the source checkpoint with the new fetch metadata and clears the previous error state

#### Scenario: Failed source sync preserves recovery context
- **WHEN** fetching or persisting a feed source fails
- **THEN** the system records the failure status and error for that source without discarding the last successful checkpoint
