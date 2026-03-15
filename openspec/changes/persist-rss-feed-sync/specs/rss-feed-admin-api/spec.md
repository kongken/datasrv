## ADDED Requirements

### Requirement: Admin API manages feed sources
The system SHALL expose admin RPCs that allow operators to create, update, enable, disable, and list managed RSS feed sources with their sync-related settings.

#### Scenario: Create feed source
- **WHEN** an operator submits a valid feed source definition with URL and display metadata
- **THEN** the admin API persists the source and returns the stored source record with a stable source ID

#### Scenario: Disable feed source
- **WHEN** an operator disables an existing feed source
- **THEN** future scheduled sync runs skip that source until it is re-enabled

### Requirement: Admin API triggers and reports sync runs
The system SHALL expose admin RPCs to trigger RSS sync for all sources or a specific source and return per-source execution results.

#### Scenario: Trigger all-source sync
- **WHEN** an operator requests a full RSS sync
- **THEN** the admin API returns a run summary containing one result per processed source

#### Scenario: Target unknown source
- **WHEN** an operator requests sync for a source ID that does not exist
- **THEN** the admin API rejects the request with a not-found error

### Requirement: Admin API exposes source status
The system SHALL expose admin RPCs that return the latest checkpoint, last success metadata, failure details, and recent sync outcome for each managed feed source.

#### Scenario: Inspect source status after failure
- **WHEN** a source has failed in a recent sync run
- **THEN** the admin API returns the last error and the last successful checkpoint for that source

#### Scenario: No RSS sources configured
- **WHEN** an operator requests RSS sync status before any source has been created
- **THEN** the admin API returns an empty source list without failing
