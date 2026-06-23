# ADR 0004: Separate persistence models from domain entities

Status: Accepted

## Context

GORM works most conveniently when its tags sit directly on the structs you persist. If those structs
are also the domain entities, the domain ends up importing GORM, which breaks the inward dependency
rule from ADR 0001 and makes the domain harder to test in isolation.

## Decision

Keep two sets of types. Domain entities (in `internal/domain`) are plain Go with unexported fields and
behaviour. Persistence models (in `internal/infrastructure/persistence/gormrepo`) are dumb structs with
GORM tags, indexes and table names. Repositories map between the two: domain to model on the way in,
model to the `Reconstitute...` constructor on the way out. Money is stored as an integer amount plus a
currency and rebuilt into the `Money` value object.

## Consequences

The domain stays free of framework concerns and fully unit-testable. Schema details (column types,
indexes, cascade rules) live in one obvious place. The cost is the mapping code and a second set of
structs that has to be kept in step with the domain. For a domain-centric service where correctness is
the priority this is a reasonable price; for a thin CRUD service it probably would not be.
