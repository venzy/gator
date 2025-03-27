# gator
Boot.dev RSS aggregator project in Go + HTTP + (Postgre)SQL

## Installing
1. You'll need Go 1.23+ and PostgreSQL 15+ installed
2. ```go install github.com/venzy/gator```
3. ```go install github.com/pressly/goose/v3/cmd/goose@latest```
4. Follow instructions in boot.dev Ch2 Database to create gator database
5. ```(cd sql/schema && goose postgres "postgres://postgres:@localhost:5432/gator" up)```
    - TODO: Proper version of this app would package the postgresql DB creation and schema handling
6. Create `~/.gatorconfig.json` with the following content:
    ```json
    {
        db_url: "postgres://postgres:@localhost:5432/gator"
    }
    ```

## Running
- `gator register <username>`
- `gator login <username>`
- `gator addFeed <name> <url>`
    - Adds feed (if not already added) to list of feeds to be aggregated
    - Auto-follows that feed for the logged-in user
- `gator agg <period>`
    - Runs infinite poll of added feeds from all users, one feed per period e.g. 60s, collecting RSS content into database
- `gator browse [row limit]`
    - Show summary of `[row limit]` (default: 2) most recent posts across all the logged in user's current feeds