## ADDED Requirements

### Requirement: Query API lists persisted feeds
The system SHALL expose a read-only query RPC that returns persisted feed sources with pagination-safe ordering for frontend consumption.

#### Scenario: List feeds with enabled and disabled sources
- **WHEN** a client requests the feed list
- **THEN** the API returns persisted feed source records with their user-facing metadata and availability state

### Requirement: Query API lists feed contents by source
The system SHALL expose a query RPC that lists persisted feed items for a selected feed source in reverse chronological order with page-based pagination.

#### Scenario: Browse latest feed contents
- **WHEN** a client requests feed contents for a source with a valid page and page size
- **THEN** the API returns items ordered from newest to oldest and indicates whether another page exists

#### Scenario: Query unknown feed source
- **WHEN** a client requests feed contents for a source ID that does not exist
- **THEN** the API returns a not-found error

### Requirement: Query API returns single feed content details
The system SHALL expose a query RPC that returns one persisted feed item with its source metadata, content fields, and publication timestamps.

#### Scenario: Read feed item details
- **WHEN** a client requests a persisted feed item by its stored item ID
- **THEN** the API returns the full item details for that record

#### Scenario: Read missing feed item
- **WHEN** a client requests a feed item ID that is not stored
- **THEN** the API returns a not-found error
